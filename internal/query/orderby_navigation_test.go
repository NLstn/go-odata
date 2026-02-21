package query

import (
	"strings"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type testOrderNav struct {
	ID         int             `json:"ID" odata:"key"`
	CustomerID int             `json:"CustomerID"`
	Customer   testCustomerNav `json:"Customer" gorm:"foreignKey:CustomerID"`
}

type testCustomerNav struct {
	ID       int           `json:"ID" odata:"key"`
	Name     string        `json:"Name"`
	RegionID int           `json:"RegionID"`
	Region   testRegionNav `json:"Region" gorm:"foreignKey:RegionID"`
}

type testRegionNav struct {
	ID   int    `json:"ID" odata:"key"`
	Name string `json:"Name" gorm:"column:region_name"`
}

func (testOrderNav) TableName() string {
	return "orders"
}

func (testCustomerNav) TableName() string {
	return "customers"
}

func (testRegionNav) TableName() string {
	return "regions"
}

func TestApplyOrderByNavigationPropertySQL(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DryRun: true})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	orderMeta, err := metadata.AnalyzeEntity(testOrderNav{})
	if err != nil {
		t.Fatalf("Failed to analyze order entity: %v", err)
	}

	customerMeta, err := metadata.AnalyzeEntity(testCustomerNav{})
	if err != nil {
		t.Fatalf("Failed to analyze customer entity: %v", err)
	}

	regionMeta, err := metadata.AnalyzeEntity(testRegionNav{})
	if err != nil {
		t.Fatalf("Failed to analyze region entity: %v", err)
	}

	orderMeta.SetEntitiesRegistry(map[string]*metadata.EntityMetadata{
		orderMeta.EntityName:    orderMeta,
		customerMeta.EntityName: customerMeta,
		regionMeta.EntityName:   regionMeta,
	})

	orderBy := []OrderByItem{
		{Property: "Customer/Name", Descending: true},
	}

	query := applyOrderBy(db.Model(&testOrderNav{}), orderBy, orderMeta)
	var orders []testOrderNav
	stmt := query.Find(&orders).Statement
	sql := stmt.SQL.String()

	expectedJoin := `LEFT JOIN "customers" AS "nav_customer" ON "orders"."customer_id" = "nav_customer"."id"`
	if !strings.Contains(sql, expectedJoin) {
		t.Fatalf("expected join clause %q in SQL: %s", expectedJoin, sql)
	}

	expectedOrder := `ORDER BY "nav_customer"."name" DESC`
	if !strings.Contains(sql, expectedOrder) {
		t.Fatalf("expected order by clause %q in SQL: %s", expectedOrder, sql)
	}
}

func TestApplyOrderByMultiHopNavigationPropertySQL(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DryRun: true})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	orderMeta, err := metadata.AnalyzeEntity(testOrderNav{})
	if err != nil {
		t.Fatalf("Failed to analyze order entity: %v", err)
	}

	customerMeta, err := metadata.AnalyzeEntity(testCustomerNav{})
	if err != nil {
		t.Fatalf("Failed to analyze customer entity: %v", err)
	}

	regionMeta, err := metadata.AnalyzeEntity(testRegionNav{})
	if err != nil {
		t.Fatalf("Failed to analyze region entity: %v", err)
	}

	orderMeta.SetEntitiesRegistry(map[string]*metadata.EntityMetadata{
		orderMeta.EntityName:    orderMeta,
		customerMeta.EntityName: customerMeta,
		regionMeta.EntityName:   regionMeta,
	})

	orderBy := []OrderByItem{
		{Property: "Customer/Region/Name", Descending: false},
	}

	query := applyOrderBy(db.Model(&testOrderNav{}), orderBy, orderMeta)
	var orders []testOrderNav
	stmt := query.Find(&orders).Statement
	sql := stmt.SQL.String()

	expectedCustomerJoin := `LEFT JOIN "customers" AS "nav_customer" ON "orders"."customer_id" = "nav_customer"."id"`
	if !strings.Contains(sql, expectedCustomerJoin) {
		t.Fatalf("expected join clause %q in SQL: %s", expectedCustomerJoin, sql)
	}

	expectedRegionJoin := `LEFT JOIN "regions" AS "nav_customer_region" ON "nav_customer"."region_id" = "nav_customer_region"."id"`
	if !strings.Contains(sql, expectedRegionJoin) {
		t.Fatalf("expected join clause %q in SQL: %s", expectedRegionJoin, sql)
	}

	expectedOrder := `ORDER BY "nav_customer_region"."region_name"`
	if !strings.Contains(sql, expectedOrder) {
		t.Fatalf("expected order by clause %q in SQL: %s", expectedOrder, sql)
	}
}

// TestApplyOrderByDeduplicatesFilterNavigationJOIN verifies that when $filter already
// added a navigation JOIN, $orderby on the same navigation path does not add it again.
func TestApplyOrderByDeduplicatesFilterNavigationJOIN(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DryRun: true})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	orderMeta, err := metadata.AnalyzeEntity(testOrderNav{})
	if err != nil {
		t.Fatalf("Failed to analyze order entity: %v", err)
	}
	customerMeta, err := metadata.AnalyzeEntity(testCustomerNav{})
	if err != nil {
		t.Fatalf("Failed to analyze customer entity: %v", err)
	}
	regionMeta, err := metadata.AnalyzeEntity(testRegionNav{})
	if err != nil {
		t.Fatalf("Failed to analyze region entity: %v", err)
	}
	orderMeta.SetEntitiesRegistry(map[string]*metadata.EntityMetadata{
		orderMeta.EntityName:    orderMeta,
		customerMeta.EntityName: customerMeta,
		regionMeta.EntityName:   regionMeta,
	})

	// Simulate $filter=Customer/Name ne '' (adds nav_customer JOIN)
	filter := &FilterExpression{
		Property: "Customer/Name",
		Operator: "ne",
		Value:    "",
	}
	db = applyFilter(db.Model(&testOrderNav{}), filter, orderMeta)

	// Simulate $orderby=Customer/Name asc (must NOT add a second nav_customer JOIN)
	orderBy := []OrderByItem{
		{Property: "Customer/Name", Descending: false},
	}
	db = applyOrderBy(db, orderBy, orderMeta)

	var orders []testOrderNav
	stmt := db.Find(&orders).Statement
	sql := stmt.SQL.String()

	joinClause := `LEFT JOIN "customers" AS "nav_customer"`
	count := strings.Count(sql, joinClause)
	if count != 1 {
		t.Fatalf("expected exactly 1 occurrence of %q in SQL, got %d:\n%s", joinClause, count, sql)
	}
}

// TestApplyOrderByRegularPropertyQualifiedWhenNavJoinPresent verifies that when a
// navigation JOIN is present, regular (non-navigation) columns in ORDER BY are
// qualified with the main table name to avoid ambiguity.
func TestApplyOrderByRegularPropertyQualifiedWhenNavJoinPresent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DryRun: true})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	orderMeta, err := metadata.AnalyzeEntity(testOrderNav{})
	if err != nil {
		t.Fatalf("Failed to analyze order entity: %v", err)
	}
	customerMeta, err := metadata.AnalyzeEntity(testCustomerNav{})
	if err != nil {
		t.Fatalf("Failed to analyze customer entity: %v", err)
	}
	regionMeta, err := metadata.AnalyzeEntity(testRegionNav{})
	if err != nil {
		t.Fatalf("Failed to analyze region entity: %v", err)
	}
	orderMeta.SetEntitiesRegistry(map[string]*metadata.EntityMetadata{
		orderMeta.EntityName:    orderMeta,
		customerMeta.EntityName: customerMeta,
		regionMeta.EntityName:   regionMeta,
	})

	// $orderby=Customer/Name asc,CustomerID asc  — mix of nav and regular properties
	orderBy := []OrderByItem{
		{Property: "Customer/Name", Descending: false},
		{Property: "CustomerID", Descending: false},
	}
	db = applyOrderBy(db.Model(&testOrderNav{}), orderBy, orderMeta)

	var orders []testOrderNav
	stmt := db.Find(&orders).Statement
	sql := stmt.SQL.String()

	// Regular column must be qualified with the main table to avoid ambiguity with joined tables
	expectedQualified := `"orders"."customer_id"`
	if !strings.Contains(sql, expectedQualified) {
		t.Fatalf("expected regular column to be qualified as %q in SQL: %s", expectedQualified, sql)
	}
}
