package query

import (
	"strings"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type testCompositeParent struct {
	OrderID int                  `json:"OrderID" gorm:"primaryKey;column:order_key" odata:"key"`
	LineID  int                  `json:"LineID" gorm:"primaryKey;column:line_key" odata:"key"`
	Items   []testCompositeChild `json:"Items" gorm:"foreignKey:OrderRef,LineRef;references:OrderID,LineID"`
}

type testCompositeChild struct {
	OrderRef int `json:"OrderRef" gorm:"column:order_ref" odata:"key"`
	LineRef  int `json:"LineRef" gorm:"column:line_ref" odata:"key"`
	Quantity int `json:"Quantity" gorm:"column:qty_value"`
}

func (testCompositeParent) TableName() string {
	return "parent_records"
}

func (testCompositeChild) TableName() string {
	return "child_records"
}

func TestBuildLambdaCondition_CompositeKeyUsesColumnNames(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DryRun: true})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	parentMeta, err := metadata.AnalyzeEntity(testCompositeParent{})
	if err != nil {
		t.Fatalf("Failed to analyze parent entity: %v", err)
	}

	childMeta, err := metadata.AnalyzeEntity(testCompositeChild{})
	if err != nil {
		t.Fatalf("Failed to analyze child entity: %v", err)
	}

	registry := map[string]*metadata.EntityMetadata{
		parentMeta.EntityName: parentMeta,
		childMeta.EntityName:  childMeta,
	}
	parentMeta.SetEntitiesRegistry(registry)
	childMeta.SetEntitiesRegistry(registry)

	filterExpr, err := parseFilter("Items/any(i: i/Quantity gt 0)", parentMeta, nil, 0)
	if err != nil {
		t.Fatalf("Failed to parse filter: %v", err)
	}

	query := ApplyFilterOnly(db.Model(&testCompositeParent{}), filterExpr, parentMeta)
	var parents []testCompositeParent
	stmt := query.Find(&parents).Statement
	sql := stmt.SQL.String()

	expectedJoin := `"child_records"."order_ref" = "parent_records"."order_key" AND "child_records"."line_ref" = "parent_records"."line_key"`
	if !strings.Contains(sql, expectedJoin) {
		t.Fatalf("expected join clause %q in SQL: %s", expectedJoin, sql)
	}
}
