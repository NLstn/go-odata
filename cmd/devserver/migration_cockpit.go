package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/NLstn/go-odata/devserver/entities"
	"gorm.io/gorm"
)

// migrationTaskStatus represents the lifecycle of an import/export job.
type migrationTaskStatus string

const (
	taskPending   migrationTaskStatus = "pending"
	taskRunning   migrationTaskStatus = "running"
	taskCompleted migrationTaskStatus = "completed"
	taskFailed    migrationTaskStatus = "failed"
)

// migrationTaskType identifies the action performed by a task.
type migrationTaskType string

const (
	taskTypeImport migrationTaskType = "import"
	taskTypeExport migrationTaskType = "export"
)

// migrationTaskRecord stores asynchronous job state in the devserver database.
type migrationTaskRecord struct {
	ID             string              `gorm:"primaryKey;size:48"`
	Type           migrationTaskType   `gorm:"size:16;not null"`
	Status         migrationTaskStatus `gorm:"size:16;not null"`
	FileName       string              `gorm:"size:255"`
	ResultPath     string              `gorm:"size:512"`
	ResultFileName string              `gorm:"size:255"`
	ResultMime     string              `gorm:"size:64"`
	Summary        string              `gorm:"size:512"`
	ErrorText      string              `gorm:"size:512"`
	CreatedAt      time.Time           `gorm:"not null"`
	UpdatedAt      time.Time           `gorm:"not null"`
	CompletedAt    *time.Time
}

func (migrationTaskRecord) TableName() string {
	return "migration_cockpit_tasks"
}

// migrationTaskDTO is exposed via the HTTP API for UI consumption.
type migrationTaskDTO struct {
	ID          string     `json:"id"`
	Type        string     `json:"type"`
	Status      string     `json:"status"`
	FileName    string     `json:"fileName,omitempty"`
	Summary     string     `json:"summary,omitempty"`
	Error       string     `json:"error,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
	DownloadURL string     `json:"downloadUrl,omitempty"`
}

// migrationCockpit orchestrates import/export jobs and exposes HTTP handlers.
type migrationCockpit struct {
	db          *gorm.DB
	pageTmpl    *template.Template
	tasksMu     sync.Mutex
	downloadMap map[string]string
}

func newMigrationCockpit(db *gorm.DB) (*migrationCockpit, error) {
	if db == nil {
		return nil, errors.New("migration cockpit requires database handle")
	}

	if err := db.AutoMigrate(&migrationTaskRecord{}); err != nil {
		return nil, fmt.Errorf("failed to migrate migration cockpit tables: %w", err)
	}

	tmpl, err := template.New("migration_cockpit").Parse(migrationCockpitHTML)
	if err != nil {
		return nil, fmt.Errorf("failed to parse migration cockpit template: %w", err)
	}

	return &migrationCockpit{
		db:          db,
		pageTmpl:    tmpl,
		downloadMap: make(map[string]string),
	}, nil
}

func (mc *migrationCockpit) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/migration-cockpit", mc.handlePage)
	mux.HandleFunc("/migration-cockpit/", mc.handleSubroutes)
}

func (mc *migrationCockpit) handlePage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := mc.pageTmpl.Execute(w, nil); err != nil {
		log.Printf("migration cockpit: failed to render template: %v", err)
	}
}

func (mc *migrationCockpit) handleSubroutes(w http.ResponseWriter, r *http.Request) {
	switch {
	case strings.HasSuffix(r.URL.Path, "/tasks") && r.Method == http.MethodGet:
		mc.handleListTasks(w, r)
		return
	case strings.HasSuffix(r.URL.Path, "/import") && r.Method == http.MethodPost:
		mc.handleImport(w, r)
		return
	case strings.HasSuffix(r.URL.Path, "/export") && r.Method == http.MethodPost:
		mc.handleExport(w, r)
		return
	case strings.HasPrefix(r.URL.Path, "/migration-cockpit/download/") && r.Method == http.MethodGet:
		mc.handleDownload(w, r)
		return
	default:
		http.NotFound(w, r)
	}
}

func (mc *migrationCockpit) handleListTasks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tasks, err := mc.loadRecentTasks(ctx, 50)
	if err != nil {
		log.Printf("migration cockpit: failed to load tasks: %v", err)
		http.Error(w, "failed to load tasks", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{"tasks": tasks}); err != nil {
		log.Printf("migration cockpit: failed to encode tasks response: %v", err)
	}
}

func (mc *migrationCockpit) handleImport(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10MB limit
		http.Error(w, "failed to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "CSV file is required", http.StatusBadRequest)
		return
	}
	defer func() {
		_ = file.Close()
	}()

	data, err := io.ReadAll(file)
	if err != nil {
		log.Printf("migration cockpit: failed to read upload: %v", err)
		http.Error(w, "failed to read uploaded file", http.StatusBadRequest)
		return
	}

	task, err := mc.startImportTask(r.Context(), header.Filename, data)
	if err != nil {
		log.Printf("migration cockpit: failed to start import: %v", err)
		http.Error(w, "failed to start import", http.StatusInternalServerError)
		return
	}

	mc.writeTaskAcceptedResponse(w, task)
}

func (mc *migrationCockpit) handleExport(w http.ResponseWriter, r *http.Request) {
	task, err := mc.startExportTask(r.Context())
	if err != nil {
		log.Printf("migration cockpit: failed to start export: %v", err)
		http.Error(w, "failed to start export", http.StatusInternalServerError)
		return
	}

	mc.writeTaskAcceptedResponse(w, task)
}

func (mc *migrationCockpit) handleDownload(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/migration-cockpit/download/")
	if id == "" {
		http.NotFound(w, r)
		return
	}

	record, err := mc.getTask(r.Context(), id)
	if err != nil {
		log.Printf("migration cockpit: failed to load task for download: %v", err)
		http.NotFound(w, r)
		return
	}

	if record.Type != taskTypeExport || record.Status != taskCompleted || record.ResultPath == "" {
		http.NotFound(w, r)
		return
	}

	mc.tasksMu.Lock()
	path := mc.downloadMap[id]
	mc.tasksMu.Unlock()
	if path == "" {
		path = record.ResultPath
	}

	if _, err := os.Stat(path); err != nil {
		http.NotFound(w, r)
		return
	}

	fileName := record.ResultFileName
	if fileName == "" {
		fileName = filepath.Base(path)
	}

	w.Header().Set("Content-Type", record.ResultMime)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	http.ServeFile(w, r, path)
}

func (mc *migrationCockpit) writeTaskAcceptedResponse(w http.ResponseWriter, task *migrationTaskRecord) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]string{"taskId": task.ID})
}

func (mc *migrationCockpit) loadRecentTasks(ctx context.Context, limit int) ([]migrationTaskDTO, error) {
	var records []migrationTaskRecord
	if err := mc.db.WithContext(ctx).Order("created_at DESC").Limit(limit).Find(&records).Error; err != nil {
		return nil, err
	}

	dtos := make([]migrationTaskDTO, 0, len(records))
	for _, rec := range records {
		dto := migrationTaskDTO{
			ID:          rec.ID,
			Type:        string(rec.Type),
			Status:      string(rec.Status),
			FileName:    rec.FileName,
			Summary:     rec.Summary,
			Error:       rec.ErrorText,
			CreatedAt:   rec.CreatedAt,
			UpdatedAt:   rec.UpdatedAt,
			CompletedAt: rec.CompletedAt,
		}
		if rec.Type == taskTypeExport && rec.Status == taskCompleted && rec.ResultPath != "" {
			dto.DownloadURL = "/migration-cockpit/download/" + rec.ID
		}
		dtos = append(dtos, dto)
	}
	return dtos, nil
}

func (mc *migrationCockpit) getTask(ctx context.Context, id string) (*migrationTaskRecord, error) {
	var record migrationTaskRecord
	if err := mc.db.WithContext(ctx).First(&record, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

func (mc *migrationCockpit) startImportTask(ctx context.Context, fileName string, data []byte) (*migrationTaskRecord, error) {
	id, err := generateTaskID()
	if err != nil {
		return nil, err
	}
	now := time.Now()
	task := &migrationTaskRecord{
		ID:        id,
		Type:      taskTypeImport,
		Status:    taskPending,
		FileName:  fileName,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := mc.db.WithContext(ctx).Create(task).Error; err != nil {
		return nil, err
	}

	go mc.runImportTask(id, fileName, data)
	return task, nil
}

func (mc *migrationCockpit) startExportTask(ctx context.Context) (*migrationTaskRecord, error) {
	id, err := generateTaskID()
	if err != nil {
		return nil, err
	}
	now := time.Now()
	task := &migrationTaskRecord{
		ID:        id,
		Type:      taskTypeExport,
		Status:    taskPending,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := mc.db.WithContext(ctx).Create(task).Error; err != nil {
		return nil, err
	}

	go mc.runExportTask(id)
	return task, nil
}

func (mc *migrationCockpit) runImportTask(id, fileName string, data []byte) {
	if err := mc.updateStatus(id, taskRunning, map[string]interface{}{}); err != nil {
		log.Printf("migration cockpit: failed to mark import running: %v", err)
	}

	summary, err := mc.performImport(data)
	if err != nil {
		if updErr := mc.updateStatus(id, taskFailed, map[string]interface{}{"error_text": err.Error(), "summary": ""}); updErr != nil {
			log.Printf("migration cockpit: failed to mark import failure: %v", updErr)
		}
		log.Printf("migration cockpit: import task failed (%s): %v", fileName, err)
		return
	}

	updates := map[string]interface{}{
		"summary": summary,
	}
	if err := mc.updateStatus(id, taskCompleted, updates); err != nil {
		log.Printf("migration cockpit: failed to mark import completed: %v", err)
	}
}

func (mc *migrationCockpit) runExportTask(id string) {
	if err := mc.updateStatus(id, taskRunning, map[string]interface{}{}); err != nil {
		log.Printf("migration cockpit: failed to mark export running: %v", err)
	}

	path, fileName, err := mc.performExport()
	if err != nil {
		if updErr := mc.updateStatus(id, taskFailed, map[string]interface{}{"error_text": err.Error(), "summary": ""}); updErr != nil {
			log.Printf("migration cockpit: failed to mark export failure: %v", updErr)
		}
		log.Printf("migration cockpit: export task failed: %v", err)
		return
	}

	mc.tasksMu.Lock()
	mc.downloadMap[id] = path
	mc.tasksMu.Unlock()

	updates := map[string]interface{}{
		"result_path":      path,
		"result_file_name": fileName,
		"result_mime":      "text/csv",
		"summary":          "Export ready",
	}
	if err := mc.updateStatus(id, taskCompleted, updates); err != nil {
		log.Printf("migration cockpit: failed to mark export completed: %v", err)
	}
}

func (mc *migrationCockpit) updateStatus(id string, status migrationTaskStatus, extra map[string]interface{}) error {
	now := time.Now()
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": now,
	}
	for k, v := range extra {
		updates[k] = v
	}
	if status == taskCompleted || status == taskFailed {
		updates["completed_at"] = &now
	}
	return mc.db.Model(&migrationTaskRecord{}).Where("id = ?", id).Updates(updates).Error
}

func (mc *migrationCockpit) performImport(data []byte) (string, error) {
	reader := csv.NewReader(bytes.NewReader(data))
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1

	rows, err := reader.ReadAll()
	if err != nil {
		return "", fmt.Errorf("failed to parse CSV: %w", err)
	}
	if len(rows) == 0 {
		return "", errors.New("CSV file is empty")
	}

	header := rows[0]
	columnIndex := make(map[string]int)
	for i, col := range header {
		columnIndex[strings.ToLower(strings.TrimSpace(col))] = i
	}

	required := []string{"name", "price"}
	for _, col := range required {
		if _, ok := columnIndex[col]; !ok {
			return "", fmt.Errorf("missing required column '%s'", col)
		}
	}

	get := func(row []string, column string) string {
		idx, ok := columnIndex[column]
		if !ok || idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}

	var createdCount, updatedCount int
	err = mc.db.Transaction(func(tx *gorm.DB) error {
		for rowIdx, row := range rows[1:] {
			if len(row) == 0 {
				continue
			}

			name := get(row, "name")
			if name == "" {
				return fmt.Errorf("row %d: product name is required", rowIdx+2)
			}

			priceStr := get(row, "price")
			if priceStr == "" {
				return fmt.Errorf("row %d: price is required", rowIdx+2)
			}
			price, err := strconv.ParseFloat(priceStr, 64)
			if err != nil {
				return fmt.Errorf("row %d: invalid price: %w", rowIdx+2, err)
			}

			var categoryID *uint
			if catStr := get(row, "categoryid"); catStr != "" {
				catVal, err := strconv.Atoi(catStr)
				if err != nil {
					return fmt.Errorf("row %d: invalid category ID: %w", rowIdx+2, err)
				}
				catUint := uint(catVal)
				categoryID = &catUint
			}

			status := entities.ProductStatusInStock
			if statusStr := get(row, "status"); statusStr != "" {
				parsed, err := strconv.Atoi(statusStr)
				if err != nil {
					return fmt.Errorf("row %d: invalid status: %w", rowIdx+2, err)
				}
				status = entities.ProductStatus(parsed)
			}

			var description *string
			if desc := get(row, "description"); desc != "" {
				description = &desc
			}

			idStr := get(row, "id")
			if idStr != "" {
				parsedID, err := strconv.Atoi(idStr)
				if err != nil {
					return fmt.Errorf("row %d: invalid ID: %w", rowIdx+2, err)
				}
				if parsedID > 0 {
					var existing entities.Product
					if err := tx.First(&existing, parsedID).Error; err == nil {
						existing.Name = name
						existing.Price = price
						existing.CategoryID = categoryID
						existing.Status = status
						existing.Description = description
						existing.Version++
						if err := tx.Save(&existing).Error; err != nil {
							return fmt.Errorf("row %d: failed to update product: %w", rowIdx+2, err)
						}
						updatedCount++
						continue
					}
				}
			}

			product := entities.Product{
				Name:        name,
				Price:       price,
				CategoryID:  categoryID,
				Status:      status,
				CreatedAt:   time.Now(),
				Version:     1,
				Description: description,
			}
			if err := tx.Create(&product).Error; err != nil {
				return fmt.Errorf("row %d: failed to create product: %w", rowIdx+2, err)
			}
			createdCount++
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Imported %d product(s), updated %d product(s)", createdCount, updatedCount), nil
}

func (mc *migrationCockpit) performExport() (string, string, error) {
	var products []entities.Product
	if err := mc.db.Order("id ASC").Find(&products).Error; err != nil {
		return "", "", fmt.Errorf("failed to load products: %w", err)
	}

	buffer := &bytes.Buffer{}
	writer := csv.NewWriter(buffer)
	header := []string{"ID", "Name", "Price", "CategoryID", "Status", "Description"}
	if err := writer.Write(header); err != nil {
		return "", "", fmt.Errorf("failed to write header: %w", err)
	}

	for _, product := range products {
		var category string
		if product.CategoryID != nil {
			category = strconv.FormatUint(uint64(*product.CategoryID), 10)
		}
		desc := ""
		if product.Description != nil {
			desc = *product.Description
		}
		record := []string{
			strconv.FormatUint(uint64(product.ID), 10),
			product.Name,
			strconv.FormatFloat(product.Price, 'f', 2, 64),
			category,
			strconv.Itoa(int(product.Status)),
			desc,
		}
		if err := writer.Write(record); err != nil {
			return "", "", fmt.Errorf("failed to write record: %w", err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", "", fmt.Errorf("failed to finalize CSV: %w", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	fileName := fmt.Sprintf("migration_products_%s.csv", timestamp)
	tempFile, err := os.CreateTemp("", "migration-export-*.csv")
	if err != nil {
		return "", "", fmt.Errorf("failed to create temp file: %w", err)
	}
	if _, err := tempFile.Write(buffer.Bytes()); err != nil {
		_ = tempFile.Close()
		return "", "", fmt.Errorf("failed to write export file: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return "", "", fmt.Errorf("failed to close export file: %w", err)
	}

	return tempFile.Name(), fileName, nil
}

func generateTaskID() (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

const migrationCockpitHTML = `<!DOCTYPE html>
<html lang="en">
<head>
        <meta charset="UTF-8">
        <title>Migration Cockpit</title>
        <meta name="viewport" content="width=device-width, initial-scale=1">
        <style>
                body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; margin: 0; padding: 0; background: #f5f6fa; color: #1f2933; }
                header { background: #243b53; color: #fff; padding: 24px; }
                main { padding: 24px; max-width: 960px; margin: 0 auto; }
                h1 { margin: 0 0 8px; font-size: 28px; }
                p { margin: 8px 0 16px; }
                section { background: #fff; border-radius: 12px; padding: 24px; margin-bottom: 24px; box-shadow: 0 10px 25px rgba(15, 23, 42, 0.05); }
                button { background: #3366ff; color: #fff; border: none; padding: 10px 18px; border-radius: 8px; cursor: pointer; font-size: 15px; }
                button:hover { background: #254eda; }
                input[type="file"] { padding: 8px 0; }
                .actions { display: flex; gap: 16px; flex-wrap: wrap; align-items: center; }
                .message { margin-top: 12px; font-weight: 600; }
                table { width: 100%; border-collapse: collapse; margin-top: 16px; }
                th, td { padding: 12px; text-align: left; border-bottom: 1px solid #e4e7eb; vertical-align: top; }
                th { background: #f0f4ff; font-size: 14px; text-transform: uppercase; letter-spacing: 0.04em; }
                .status { display: inline-flex; align-items: center; gap: 6px; padding: 4px 10px; border-radius: 999px; font-size: 13px; font-weight: 600; }
                .status.pending { background: #fdf2e9; color: #ad5200; }
                .status.running { background: #eff6ff; color: #1d4ed8; }
                .status.completed { background: #ecfdf5; color: #047857; }
                .status.failed { background: #fef2f2; color: #b91c1c; }
                .small { font-size: 13px; color: #52606d; }
                .summary { margin-top: 4px; font-size: 14px; color: #364152; }
                .tasks-empty { text-align: center; padding: 32px; color: #7b8794; }
                @media (max-width: 720px) {
                        table, thead, tbody, th, td, tr { display: block; }
                        thead tr { display: none; }
                        tr { margin-bottom: 12px; background: #fff; border-radius: 12px; padding: 12px; box-shadow: 0 6px 16px rgba(15, 23, 42, 0.05); }
                        td { border: none; padding: 8px 0; }
                        td::before { content: attr(data-label); font-weight: 600; display: block; margin-bottom: 4px; color: #486581; }
                }
        </style>
</head>
<body>
        <header>
                <h1>Migration Cockpit</h1>
                <p>Import and export product data using background jobs. Tasks run asynchronously so you can monitor progress without blocking.</p>
        </header>
        <main>
                <section>
                        <h2>Import products from CSV</h2>
                        <p class="small">Required columns: <strong>Name</strong>, <strong>Price</strong>. Optional: ID, CategoryID, Status, Description.</p>
                        <form id="importForm" class="actions">
                                <input type="file" id="importFile" name="file" accept=".csv" required>
                                <button type="submit">Start Import</button>
                                <span id="importMessage" class="message"></span>
                        </form>
                </section>
                <section>
                        <h2>Export current products</h2>
                        <p class="small">Generates a CSV snapshot of all products. The download link appears once the job finishes.</p>
                        <div class="actions">
                                <button id="exportButton">Start Export</button>
                                <span id="exportMessage" class="message"></span>
                        </div>
                </section>
                <section>
                        <h2>Recent tasks</h2>
                        <div id="tasksContainer"></div>
                </section>
        </main>
        <script>
        const tasksContainer = document.getElementById('tasksContainer');
        const importForm = document.getElementById('importForm');
        const importMessage = document.getElementById('importMessage');
        const exportButton = document.getElementById('exportButton');
        const exportMessage = document.getElementById('exportMessage');

        function setMessage(el, text, isError = false) {
                el.textContent = text;
                el.style.color = isError ? '#b91c1c' : '#1f2933';
        }

        importForm.addEventListener('submit', async (event) => {
                event.preventDefault();
                const fileInput = document.getElementById('importFile');
                if (!fileInput.files.length) {
                        setMessage(importMessage, 'Please choose a CSV file.', true);
                        return;
                }
                const formData = new FormData();
                formData.append('file', fileInput.files[0]);
                setMessage(importMessage, 'Starting import…');
                try {
                        const response = await fetch('/migration-cockpit/import', {
                                method: 'POST',
                                body: formData
                        });
                        if (!response.ok) {
                                const text = await response.text();
                                throw new Error(text || 'Failed to start import');
                        }
                        const data = await response.json();
                        setMessage(importMessage, 'Import task queued (ID: ' + data.taskId + ').');
                        fileInput.value = '';
                        await refreshTasks();
                } catch (err) {
                        setMessage(importMessage, err.message, true);
                }
        });

        exportButton.addEventListener('click', async () => {
                setMessage(exportMessage, 'Starting export…');
                try {
                        const response = await fetch('/migration-cockpit/export', { method: 'POST' });
                        if (!response.ok) {
                                const text = await response.text();
                                throw new Error(text || 'Failed to start export');
                        }
                        const data = await response.json();
                        setMessage(exportMessage, 'Export task queued (ID: ' + data.taskId + ').');
                        await refreshTasks();
                } catch (err) {
                        setMessage(exportMessage, err.message, true);
                }
        });

        function statusBadge(status) {
                const normalized = status.toLowerCase();
                return '<span class="status ' + normalized + '">' + normalized + '</span>';
        }

        function renderTasks(tasks) {
                if (!tasks.length) {
                        tasksContainer.innerHTML = '<div class="tasks-empty">No tasks yet. Start an import or export to see progress.</div>';
                        return;
                }
                const rows = tasks.map(task => {
                        const created = new Date(task.createdAt).toLocaleString();
                        const updated = new Date(task.updatedAt).toLocaleString();
                        const completed = task.completedAt ? new Date(task.completedAt).toLocaleString() : '—';
                        const summary = task.summary ? '<div class="summary">' + task.summary + '</div>' : '';
                        const error = task.error ? '<div class="summary" style="color:#b91c1c;">' + task.error + '</div>' : '';
                        const download = task.downloadUrl ? '<a href="' + task.downloadUrl + '">Download CSV</a>' : '';
                        return '<tr>' +
                                '<td data-label="Task">' + task.id + '<div class="small">' + task.type + '</div></td>' +
                                '<td data-label="Status">' + statusBadge(task.status) + summary + error + '</td>' +
                                '<td data-label="Created" class="small">' + created + '</td>' +
                                '<td data-label="Updated" class="small">' + updated + '</td>' +
                                '<td data-label="Completed" class="small">' + completed + '</td>' +
                                '<td data-label="Result" class="small">' + download + '</td>' +
                        '</tr>';
                }).join('');
                tasksContainer.innerHTML = '<table>' +
                        '<thead>' +
                                '<tr>' +
                                        '<th>Task</th>' +
                                        '<th>Status</th>' +
                                        '<th>Created</th>' +
                                        '<th>Updated</th>' +
                                        '<th>Completed</th>' +
                                        '<th>Result</th>' +
                                '</tr>' +
                        '</thead>' +
                        '<tbody>' + rows + '</tbody>' +
                '</table>';
        }

        async function refreshTasks() {
                try {
                        const response = await fetch('/migration-cockpit/tasks');
                        if (!response.ok) {
                                throw new Error('Failed to load tasks');
                        }
                        const data = await response.json();
                        const tasks = Array.isArray(data.tasks) ? data.tasks : [];
                        tasks.sort((a, b) => new Date(b.createdAt) - new Date(a.createdAt));
                        renderTasks(tasks);
                } catch (err) {
                        tasksContainer.innerHTML = '<div class="tasks-empty">' + err.message + '</div>';
                }
        }

        refreshTasks();
        setInterval(refreshTasks, 5000);
        </script>
</body>
</html>`
