package query

import (
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type ApplierTestEntity struct {
	ID   int    `json:"id" gorm:"primarykey" odata:"key"`
	Name string `json:"name"`
}

func setupApplierTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&ApplierTestEntity{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	return db
}

func TestApplyOffsetWithLimit(t *testing.T) {
	db := setupApplierTestDB(t)

	tests := []struct {
		name string
		skip int
		top  *int
	}{
		{
			name: "With skip and no top",
			skip: 10,
			top:  nil,
		},
		{
			name: "With skip and top",
			skip: 5,
			top:  func() *int { v := 20; return &v }(),
		},
		{
			name: "Zero skip with no top",
			skip: 0,
			top:  nil,
		},
		{
			name: "Large skip with no top",
			skip: 1000,
			top:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := applyOffsetWithLimit(db, tt.skip, tt.top)
			if result == nil {
				t.Error("applyOffsetWithLimit returned nil")
				return
			}

			// Verify that the function returns a valid *gorm.DB
			if result.Statement == nil {
				t.Error("Returned DB has nil Statement")
			}
		})
	}
}

func TestShouldUseMapResults(t *testing.T) {
	tests := []struct {
		name     string
		options  *QueryOptions
		expected bool
	}{
		{
			name:     "Nil options",
			options:  nil,
			expected: false,
		},
		{
			name:     "Empty options",
			options:  &QueryOptions{},
			expected: false,
		},
		{
			name: "With Apply transformations",
			options: &QueryOptions{
				Apply: []ApplyTransformation{
					{Type: ApplyTypeGroupBy},
				},
			},
			expected: true,
		},
		{
			name: "With Compute",
			options: &QueryOptions{
				Compute: &ComputeTransformation{
					Expressions: []ComputeExpression{
						{Alias: "total"},
					},
				},
			},
			expected: true,
		},
		{
			name: "With both Apply and Compute",
			options: &QueryOptions{
				Apply: []ApplyTransformation{
					{Type: ApplyTypeGroupBy},
				},
				Compute: &ComputeTransformation{
					Expressions: []ComputeExpression{
						{Alias: "total"},
					},
				},
			},
			expected: true,
		},
		{
			name: "With only Filter",
			options: &QueryOptions{
				Filter: &FilterExpression{
					Property: "Name",
					Operator: OpEqual,
					Value:    "test",
				},
			},
			expected: false,
		},
		{
			name: "With only OrderBy",
			options: &QueryOptions{
				OrderBy: []OrderByItem{
					{Property: "Name", Descending: false},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldUseMapResults(tt.options)
			if result != tt.expected {
				t.Errorf("ShouldUseMapResults() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestApplyQueryOptions(t *testing.T) {
	db := setupApplierTestDB(t)
	meta, err := metadata.AnalyzeEntity(ApplierTestEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	// Insert test data
	testData := []ApplierTestEntity{
		{ID: 1, Name: "Alice"},
		{ID: 2, Name: "Bob"},
		{ID: 3, Name: "Charlie"},
	}
	for _, entity := range testData {
		db.Create(&entity)
	}

	tests := []struct {
		name    string
		options *QueryOptions
		wantErr bool
	}{
		{
			name:    "Nil options",
			options: nil,
			wantErr: false,
		},
		{
			name:    "Empty options",
			options: &QueryOptions{},
			wantErr: false,
		},
		{
			name: "With Top",
			options: &QueryOptions{
				Top: func() *int { v := 2; return &v }(),
			},
			wantErr: false,
		},
		{
			name: "With Skip",
			options: &QueryOptions{
				Skip: func() *int { v := 1; return &v }(),
			},
			wantErr: false,
		},
		{
			name: "With Count",
			options: &QueryOptions{
				Count: true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ApplyQueryOptions(db, tt.options, meta)
			if result == nil {
				t.Error("ApplyQueryOptions returned nil")
				return
			}

			// Try to execute the query to ensure it's valid
			var entities []ApplierTestEntity
			err := result.Find(&entities).Error
			if (err != nil) != tt.wantErr {
				t.Errorf("Query execution error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestApplyQueryOptionsWithFTS(t *testing.T) {
	db := setupApplierTestDB(t)
	meta, err := metadata.AnalyzeEntity(ApplierTestEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	// Insert test data
	testData := []ApplierTestEntity{
		{ID: 1, Name: "Alice"},
		{ID: 2, Name: "Bob"},
		{ID: 3, Name: "Charlie"},
	}
	for _, entity := range testData {
		db.Create(&entity)
	}

	tests := []struct {
		name       string
		options    *QueryOptions
		ftsManager *FTSManager
		tableName  string
	}{
		{
			name:       "Nil FTS manager",
			options:    &QueryOptions{},
			ftsManager: nil,
			tableName:  "applier_test_entities",
		},
		{
			name:       "Nil options with FTS",
			options:    nil,
			ftsManager: nil,
			tableName:  "",
		},
		{
			name: "With search but no FTS",
			options: &QueryOptions{
				Search: "test",
			},
			ftsManager: nil,
			tableName:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ApplyQueryOptionsWithFTS(db, tt.options, meta, tt.ftsManager, tt.tableName)
			if result == nil {
				t.Error("ApplyQueryOptionsWithFTS returned nil")
				return
			}

			// Verify query is valid
			var entities []ApplierTestEntity
			err := result.Find(&entities).Error
			if err != nil {
				t.Errorf("Query execution error: %v", err)
			}
		})
	}
}

func TestApplyExpandOnly(t *testing.T) {
	db := setupApplierTestDB(t)
	meta, err := metadata.AnalyzeEntity(ApplierTestEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	tests := []struct {
		name   string
		expand []ExpandOption
	}{
		{
			name:   "Nil expand",
			expand: nil,
		},
		{
			name:   "Empty expand",
			expand: []ExpandOption{},
		},
		{
			name: "With expand options",
			expand: []ExpandOption{
				{NavigationProperty: "Items"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ApplyExpandOnly(db, tt.expand, meta)
			if result == nil {
				t.Error("ApplyExpandOnly returned nil")
			}
		})
	}
}
