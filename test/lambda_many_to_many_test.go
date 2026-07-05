package odata_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// RelationProduct/RelationDescription form a self-referential many-to-many navigation property
// (RelatedProducts) plus a one-to-many navigation property (Descriptions), used to
// exercise a lambda filter nested inside a many-to-many lambda filter, i.e.
// RelatedProducts/any(r: r/Descriptions/any(d: ...)).
type RelationProduct struct {
	ID              uint                  `json:"ID" gorm:"primaryKey" odata:"key"`
	Name            string                `json:"Name"`
	Descriptions    []RelationDescription `json:"Descriptions" gorm:"foreignKey:ProductID"`
	RelatedProducts []RelationProduct     `json:"RelatedProducts,omitempty" gorm:"many2many:relation_product_relations;"`
}

type RelationDescription struct {
	ID          uint   `json:"ID" gorm:"primaryKey" odata:"key"`
	ProductID   uint   `json:"ProductID"`
	LanguageKey string `json:"LanguageKey"`
}

func setupLambdaManyToManyTest(t *testing.T) *odata.Service {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	if err := db.AutoMigrate(&RelationProduct{}, &RelationDescription{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Product 1 (Laptop) has an EN description.
	db.Create(&RelationProduct{ID: 1, Name: "Laptop"})
	db.Create(&RelationDescription{ID: 1, ProductID: 1, LanguageKey: "EN"})

	// Product 2 (Mouse) has only a DE description.
	db.Create(&RelationProduct{ID: 2, Name: "Mouse"})
	db.Create(&RelationDescription{ID: 2, ProductID: 2, LanguageKey: "DE"})

	// Product 3 (Chair) is related to Product 1 (Laptop), which has an EN description.
	db.Create(&RelationProduct{ID: 3, Name: "Chair"})
	if err := db.Exec("INSERT INTO relation_product_relations (relation_product_id, related_product_id) VALUES (3, 1)").Error; err != nil {
		t.Fatalf("Failed to seed product_relations: %v", err)
	}

	// Product 4 (Desk) is related to Product 2 (Mouse), which has no EN description.
	db.Create(&RelationProduct{ID: 4, Name: "Desk"})
	if err := db.Exec("INSERT INTO relation_product_relations (relation_product_id, related_product_id) VALUES (4, 2)").Error; err != nil {
		t.Fatalf("Failed to seed product_relations: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	if err := service.RegisterEntity(&RelationProduct{}); err != nil {
		t.Fatalf("Failed to register RelationProduct entity: %v", err)
	}
	if err := service.RegisterEntity(&RelationDescription{}); err != nil {
		t.Fatalf("Failed to register RelationDescription entity: %v", err)
	}

	return service
}

func queryRelationProductIDs(t *testing.T, service *odata.Service, filterQuery string) []float64 {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/RelationProducts?$filter="+url.QueryEscape(filterQuery), nil)
	rec := httptest.NewRecorder()
	service.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("Response missing 'value' array")
	}

	ids := make([]float64, 0, len(value))
	for _, item := range value {
		product := item.(map[string]interface{})
		ids = append(ids, product["ID"].(float64))
	}
	return ids
}

// TestLambdaManyToMany_SimplePredicate reproduces the base case from #783: a lambda
// filter directly over a many-to-many self-referential navigation property must join
// through the bridge table rather than assume a direct foreign key column exists on
// the related table.
func TestLambdaManyToMany_SimplePredicate(t *testing.T) {
	service := setupLambdaManyToManyTest(t)

	ids := queryRelationProductIDs(t, service, "RelatedProducts/any(r: r/Name eq 'Laptop')")

	if len(ids) != 1 || ids[0] != 3 {
		t.Fatalf("Expected only Chair (3), got %v", ids)
	}
}

// TestLambdaManyToMany_NestedLambda reproduces #783: a two-level nested lambda filter
// where the outer lambda ranges over a many-to-many self-referential navigation
// property (RelatedProducts) and the inner lambda ranges over a one-to-many navigation
// property (Descriptions) on the related entity must return 200 with the correct
// result, not a 500 ("no such column") caused by assuming a direct foreign key column
// exists for the many-to-many relation.
func TestLambdaManyToMany_NestedLambda(t *testing.T) {
	service := setupLambdaManyToManyTest(t)

	ids := queryRelationProductIDs(t, service,
		"RelatedProducts/any(r: r/Descriptions/any(d: d/LanguageKey eq 'EN'))")

	if len(ids) != 1 || ids[0] != 3 {
		t.Fatalf("Expected only Chair (3), got %v", ids)
	}
}

// TestLambdaManyToMany_NestedLambda_NoMatchIsVacuouslyFalse verifies that when the
// related product does not satisfy the inner predicate (Desk -> Mouse, which has no EN
// description), the outer lambda correctly excludes it rather than matching due to a
// mis-correlated subquery.
func TestLambdaManyToMany_NestedLambda_NoMatchIsVacuouslyFalse(t *testing.T) {
	service := setupLambdaManyToManyTest(t)

	ids := queryRelationProductIDs(t, service,
		"RelatedProducts/any(r: r/Descriptions/any(d: d/LanguageKey eq 'ZZ'))")

	if len(ids) != 0 {
		t.Fatalf("Expected no products to match, got %v", ids)
	}
}
