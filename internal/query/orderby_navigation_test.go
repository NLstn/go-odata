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
	ID   int    `json:"ID" odata:"key"`
	Name string `json:"Name"`
}

func (testOrderNav) TableName() string {
	return "orders"
}

func (testCustomerNav) TableName() string {
	return "customers"
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

	orderMeta.SetEntitiesRegistry(map[string]*metadata.EntityMetadata{
		orderMeta.EntityName:    orderMeta,
		customerMeta.EntityName: customerMeta,
	})

	orderBy := []OrderByItem{
		{Property: "Customer/Name", Descending: true},
	}

	query := applyOrderBy(db.Model(&testOrderNav{}), orderBy, orderMeta)
	var orders []testOrderNav
	stmt := query.Find(&orders).Statement
	sql := stmt.SQL.String()

	expectedJoin := `LEFT JOIN "customers" ON "orders"."customer_id" = "customers"."id"`
	if !strings.Contains(sql, expectedJoin) {
		t.Fatalf("expected join clause %q in SQL: %s", expectedJoin, sql)
	}

	expectedOrder := `ORDER BY "customers"."name" DESC`
	if !strings.Contains(sql, expectedOrder) {
		t.Fatalf("expected order by clause %q in SQL: %s", expectedOrder, sql)
	}
}
