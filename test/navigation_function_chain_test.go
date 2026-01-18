package odata_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type NavigationFunctionStore struct {
	ID            uint                     `json:"ID" gorm:"primaryKey" odata:"key"`
	Name          string                   `json:"Name"`
	FeaturedItems []NavigationFunctionItem `json:"FeaturedItems" gorm:"foreignKey:StoreID"`
}

type NavigationFunctionItem struct {
	ID      uint                     `json:"ID" gorm:"primaryKey" odata:"key"`
	Name    string                   `json:"Name"`
	Price   float64                  `json:"Price"`
	StoreID uint                     `json:"StoreID"`
	Store   *NavigationFunctionStore `json:"Store,omitempty" gorm:"foreignKey:StoreID"`
}

func TestBoundFunctionThroughRenamedNavigation(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&NavigationFunctionStore{}, &NavigationFunctionItem{}); err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	store := NavigationFunctionStore{ID: 1, Name: "Downtown"}
	items := []NavigationFunctionItem{
		{ID: 1, Name: "Laptop", Price: 1000, StoreID: 1},
		{ID: 2, Name: "Mouse", Price: 25, StoreID: 1},
	}
	if err := db.Create(&store).Error; err != nil {
		t.Fatalf("failed to seed store: %v", err)
	}
	if err := db.Create(&items).Error; err != nil {
		t.Fatalf("failed to seed items: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
	if err := service.RegisterEntity(&NavigationFunctionStore{}); err != nil {
		t.Fatalf("failed to register store entity: %v", err)
	}
	if err := service.RegisterEntity(&NavigationFunctionItem{}); err != nil {
		t.Fatalf("failed to register item entity: %v", err)
	}

	err = service.RegisterFunction(odata.FunctionDefinition{
		Name:       "GetAveragePrice",
		IsBound:    true,
		EntitySet:  "NavigationFunctionItems",
		Parameters: []odata.ParameterDefinition{},
		ReturnType: reflect.TypeOf(float64(0)),
		Handler: func(http.ResponseWriter, *http.Request, interface{}, map[string]interface{}) (interface{}, error) {
			var avg float64
			if err := db.Model(&NavigationFunctionItem{}).
				Where("store_id = ?", store.ID).
				Select("avg(price)").
				Scan(&avg).Error; err != nil {
				return nil, err
			}
			return avg, nil
		},
	})
	if err != nil {
		t.Fatalf("failed to register function: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/NavigationFunctionStores(1)/FeaturedItems/GetAveragePrice()", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	value, ok := payload["value"].(float64)
	if !ok {
		t.Fatalf("expected numeric value in response, got %v", payload["value"])
	}

	expected := (items[0].Price + items[1].Price) / 2
	if value != expected {
		t.Fatalf("expected average %v, got %v", expected, value)
	}
}
