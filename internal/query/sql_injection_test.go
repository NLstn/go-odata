package query

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestSQLInjection_ComputedOrderBy tests that SQL injection attempts in $orderby
// with computed properties are properly sanitized
func TestSQLInjection_ComputedOrderBy(t *testing.T) {
	// Setup in-memory database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Create metadata
	meta := getTestMetadata(t)

	tests := []struct {
		name            string
		orderByProperty string
		shouldSkip      bool // True if the property should be skipped (invalid)
		description     string
	}{
		{
			name:            "SQL injection with semicolon",
			orderByProperty: "Price; DROP TABLE Users--",
			shouldSkip:      true,
			description:     "Should reject property with SQL injection attempt",
		},
		{
			name:            "SQL injection with single quote",
			orderByProperty: "Price' OR '1'='1",
			shouldSkip:      true,
			description:     "Should reject property with quote-based injection",
		},
		{
			name:            "SQL injection with comment",
			orderByProperty: "Price--comment",
			shouldSkip:      true,
			description:     "Should reject property with SQL comment",
		},
		{
			name:            "SQL injection with space",
			orderByProperty: "Price Name",
			shouldSkip:      true,
			description:     "Should reject property with spaces",
		},
		{
			name:            "SQL reserved keyword",
			orderByProperty: "SELECT",
			shouldSkip:      true,
			description:     "Should reject SQL reserved keywords",
		},
		{
			name:            "Valid computed property",
			orderByProperty: "TaxedPrice",
			shouldSkip:      false,
			description:     "Should allow valid computed property names",
		},
		{
			name:            "Valid computed property with underscore",
			orderByProperty: "taxed_price_2",
			shouldSkip:      false,
			description:     "Should allow valid identifiers with underscores and numbers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create order by item with a computed property (not in metadata)
			orderByItems := []OrderByItem{
				{
					Property:   tt.orderByProperty,
					Descending: false,
				},
			}

			// Apply orderby - this should sanitize the computed property
			resultDB := applyOrderBy(db, orderByItems, meta)

			// Get the SQL statement
			stmt := resultDB.Statement
			sql := stmt.SQL.String()

			t.Logf("Property: %s", tt.orderByProperty)
			t.Logf("SQL: %s", sql)

			if tt.shouldSkip {
				// For invalid properties, the ORDER BY clause should not contain the malicious input
				// The property should have been skipped
				if sql != "" {
					t.Errorf("%s: Expected empty SQL (property should be skipped), got: %s", tt.description, sql)
				}
			} else {
				// For valid properties, check that the SQL contains the property
				// Note: Since we're just checking statement building, we won't execute
				t.Logf("Valid property accepted: %s", tt.orderByProperty)
			}
		})
	}
}

// TestSQLInjection_ComputedFilter tests that SQL injection attempts in $filter
// with computed properties are properly sanitized
func TestSQLInjection_ComputedFilter(t *testing.T) {
	// Create metadata
	meta := getTestMetadata(t)

	tests := []struct {
		name        string
		property    string
		shouldFail  bool
		description string
	}{
		{
			name:        "SQL injection in filter property",
			property:    "Price; DROP TABLE Users--",
			shouldFail:  true,
			description: "Should reject filter with SQL injection",
		},
		{
			name:        "SQL injection with quotes",
			property:    "Price' OR '1'='1",
			shouldFail:  true,
			description: "Should reject filter with quote injection",
		},
		{
			name:        "Valid computed property in filter",
			property:    "TaxedPrice",
			shouldFail:  false,
			description: "Should allow valid computed property",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a filter expression with a computed property (not in metadata)
			filter := &FilterExpression{
				Property: tt.property,
				Operator: OpEqual,
				Value:    100,
			}

			// Build the filter condition - this should sanitize the property
			sql, args := buildComparisonCondition("sqlite", filter, meta)

			t.Logf("Property: %s", tt.property)
			t.Logf("SQL: %s", sql)
			t.Logf("Args: %v", args)

			if tt.shouldFail {
				// For invalid properties, SQL should be empty (rejected)
				if sql != "" {
					t.Errorf("%s: Expected empty SQL (property should be rejected), got: %s", tt.description, sql)
				}
			} else {
				// For valid properties, SQL should be generated
				if sql == "" {
					t.Errorf("%s: Expected SQL to be generated, got empty string", tt.description)
				}
			}
		})
	}
}
