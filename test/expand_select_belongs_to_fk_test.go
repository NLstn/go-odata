package odata_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type ExpandSelectBelongsToAuthor struct {
	ID   uint   `json:"id" gorm:"primaryKey" odata:"key"`
	Name string `json:"name"`
}

type ExpandSelectBelongsToBook struct {
	ID       uint                         `json:"id" gorm:"primaryKey" odata:"key"`
	Title    string                       `json:"title"`
	AuthorID uint                         `json:"author_id"`
	Author   *ExpandSelectBelongsToAuthor `json:"author,omitempty" gorm:"foreignKey:AuthorID"`
}

func TestExpandWithSelectAutoIncludesForeignKeyForBelongsTo(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&ExpandSelectBelongsToAuthor{}, &ExpandSelectBelongsToBook{}); err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	author := ExpandSelectBelongsToAuthor{ID: 1, Name: "Jane"}
	book := ExpandSelectBelongsToBook{ID: 1, Title: "Go OData", AuthorID: 1}

	if err := db.Create(&author).Error; err != nil {
		t.Fatalf("failed to create author: %v", err)
	}
	if err := db.Create(&book).Error; err != nil {
		t.Fatalf("failed to create book: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	if err := service.RegisterEntity(&ExpandSelectBelongsToAuthor{}); err != nil {
		t.Fatalf("failed to register author entity: %v", err)
	}
	if err := service.RegisterEntity(&ExpandSelectBelongsToBook{}); err != nil {
		t.Fatalf("failed to register book entity: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/ExpandSelectBelongsToBooks?$select=title&$expand=author($select=name)", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	value, ok := response["value"].([]interface{})
	if !ok || len(value) == 0 {
		t.Fatalf("expected value array with at least one item")
	}

	item, ok := value[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected first item to be object")
	}

	authorValue, exists := item["author"]
	if !exists {
		t.Fatalf("expected expanded author property")
	}
	if authorValue == nil {
		t.Fatalf("expected expanded author to be non-null")
	}

	authorObj, ok := authorValue.(map[string]interface{})
	if !ok {
		t.Fatalf("expected expanded author object, got %T", authorValue)
	}
	if _, ok := authorObj["name"]; !ok {
		t.Fatalf("expected expanded author to contain selected name property")
	}
}
