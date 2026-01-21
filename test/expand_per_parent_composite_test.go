package odata_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type CompositeParent struct {
	Region   string           `json:"Region" gorm:"primaryKey" odata:"key"`
	Code     int              `json:"Code" gorm:"primaryKey" odata:"key"`
	Name     string           `json:"Name"`
	Children []CompositeChild `json:"Children" gorm:"foreignKey:ParentRegion,ParentCode;references:Region,Code"`
}

type CompositeChild struct {
	ID           int    `json:"ID" gorm:"primaryKey" odata:"key"`
	ParentRegion string `json:"ParentRegion"`
	ParentCode   int    `json:"ParentCode"`
	Detail       string `json:"Detail"`
}

type queryCountingLogger struct {
	mu          sync.Mutex
	selectCount int
}

func (l *queryCountingLogger) LogMode(level logger.LogLevel) logger.Interface {
	return l
}

func (l *queryCountingLogger) Info(ctx context.Context, msg string, data ...interface{}) {}

func (l *queryCountingLogger) Warn(ctx context.Context, msg string, data ...interface{}) {}

func (l *queryCountingLogger) Error(ctx context.Context, msg string, data ...interface{}) {}

func (l *queryCountingLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	sql, _ := fc()
	if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(sql)), "SELECT") {
		l.mu.Lock()
		l.selectCount++
		l.mu.Unlock()
	}
}

func (l *queryCountingLogger) Reset() {
	l.mu.Lock()
	l.selectCount = 0
	l.mu.Unlock()
}

func (l *queryCountingLogger) SelectCount() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.selectCount
}

func TestPerParentExpandCompositeKeysBatched(t *testing.T) {
	countLogger := &queryCountingLogger{}
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: countLogger})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&CompositeParent{}, &CompositeChild{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	parents := []CompositeParent{
		{Region: "US", Code: 1, Name: "North America"},
		{Region: "EU", Code: 2, Name: "Europe"},
	}
	if err := db.Create(&parents).Error; err != nil {
		t.Fatalf("Failed to seed parents: %v", err)
	}

	children := []CompositeChild{
		{ID: 1, ParentRegion: "US", ParentCode: 1, Detail: "First"},
		{ID: 2, ParentRegion: "US", ParentCode: 1, Detail: "Second"},
		{ID: 3, ParentRegion: "EU", ParentCode: 2, Detail: "Third"},
		{ID: 4, ParentRegion: "EU", ParentCode: 2, Detail: "Fourth"},
	}
	if err := db.Create(&children).Error; err != nil {
		t.Fatalf("Failed to seed children: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	if err := service.RegisterEntity(&CompositeParent{}); err != nil {
		t.Fatalf("Failed to register parent entity: %v", err)
	}
	if err := service.RegisterEntity(&CompositeChild{}); err != nil {
		t.Fatalf("Failed to register child entity: %v", err)
	}

	countLogger.Reset()

	// Per OData v4.01 spec section 5.1.2, semicolon (;) separates nested query options within $expand
	// Semicolons must be URL-encoded as %3B in URLs
	req := httptest.NewRequest(http.MethodGet, "/CompositeParents?$expand=Children($orderby=ID%3B$top=1)", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	values, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("Expected value array in response")
	}
	if len(values) != 2 {
		t.Fatalf("Expected 2 parents, got %d", len(values))
	}

	for _, item := range values {
		parent, ok := item.(map[string]interface{})
		if !ok {
			t.Fatal("Expected parent object in response")
		}

		childrenValue, ok := parent["Children"].([]interface{})
		if !ok {
			t.Fatal("Expected Children to be expanded")
		}
		if len(childrenValue) != 1 {
			t.Fatalf("Expected 1 child per parent after $top, got %d", len(childrenValue))
		}

		child := childrenValue[0].(map[string]interface{})
		region := parent["Region"].(string)
		expectedID := float64(1)
		if region == "EU" {
			expectedID = float64(3)
		}
		if child["ID"] != expectedID {
			t.Fatalf("Expected child ID %.0f for region %s, got %v", expectedID, region, child["ID"])
		}
	}

	if countLogger.SelectCount() != 2 {
		t.Fatalf("Expected 2 select queries (parents + batched children), got %d", countLogger.SelectCount())
	}
}
