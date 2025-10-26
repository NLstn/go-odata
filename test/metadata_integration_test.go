package odata

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Test entities with rich metadata
type Customer struct {
	ID        int       `json:"id" gorm:"primarykey" odata:"key"`
	Name      string    `json:"name" odata:"required,maxlength=100"`
	Email     *string   `json:"email" odata:"maxlength=255,nullable"`
	Phone     string    `json:"phone" odata:"maxlength=20"`
	CreatedAt time.Time `json:"createdAt"`
	Orders    []Order   `json:"orders" gorm:"foreignKey:CustomerID;references:ID"`
}

type Order struct {
	ID          int         `json:"id" gorm:"primarykey" odata:"key"`
	OrderNumber string      `json:"orderNumber" odata:"required,maxlength=50"`
	CustomerID  int         `json:"customerId" odata:"required"`
	Customer    *Customer   `json:"customer" gorm:"foreignKey:CustomerID;references:ID"`
	TotalAmount float64     `json:"totalAmount" odata:"precision=10,scale=2"`
	Status      string      `json:"status" odata:"default=pending,maxlength=20"`
	OrderDate   time.Time   `json:"orderDate"`
	Items       []OrderItem `json:"items" gorm:"foreignKey:OrderID;references:ID"`
}

type OrderItem struct {
	ID        int      `json:"id" gorm:"primarykey" odata:"key"`
	OrderID   int      `json:"orderId" odata:"required"`
	Order     *Order   `json:"order" gorm:"foreignKey:OrderID;references:ID"`
	ProductID int      `json:"productId" odata:"required"`
	Product   *Product `json:"product" gorm:"foreignKey:ProductID;references:ID"`
	Quantity  int      `json:"quantity" odata:"required"`
	UnitPrice float64  `json:"unitPrice" odata:"precision=10,scale=2"`
}

type Product struct {
	ID          int     `json:"id" gorm:"primarykey" odata:"key"`
	Name        string  `json:"name" odata:"required,maxlength=100"`
	Description *string `json:"description" odata:"maxlength=1000,nullable"`
	SKU         string  `json:"sku" odata:"maxlength=50,default=AUTO"`
	Price       float64 `json:"price" odata:"precision=10,scale=2"`
	Stock       int     `json:"stock" odata:"default=0"`
	Active      bool    `json:"active" odata:"default=true"`
}

func setupTestServer(t *testing.T) *httptest.Server {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto-migrate all tables
	if err := db.AutoMigrate(&Customer{}, &Order{}, &OrderItem{}, &Product{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Create OData service
	service := odata.NewService(db)
	_ = service.RegisterEntity(&Customer{})
	_ = service.RegisterEntity(&Order{})
	_ = service.RegisterEntity(&OrderItem{})
	_ = service.RegisterEntity(&Product{})

	// Create test server
	server := httptest.NewServer(service)

	return server
}

func TestMetadataIntegrationXML(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	resp, err := http.Get(server.URL + "/$metadata")
	if err != nil {
		t.Fatalf("Failed to get metadata: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status = %v, want %v", resp.StatusCode, http.StatusOK)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	bodyStr := string(body)

	// Verify XML structure
	if !strings.Contains(bodyStr, "<?xml") {
		t.Error("Response should be XML")
	}

	// Verify entity types are present
	if !strings.Contains(bodyStr, `EntityType Name="Customer"`) {
		t.Error("XML should contain Customer entity type")
	}
	if !strings.Contains(bodyStr, `EntityType Name="Order"`) {
		t.Error("XML should contain Order entity type")
	}
	if !strings.Contains(bodyStr, `EntityType Name="Product"`) {
		t.Error("XML should contain Product entity type")
	}

	// Verify facets
	if !strings.Contains(bodyStr, `MaxLength="100"`) {
		t.Error("XML should contain MaxLength facet")
	}
	if !strings.Contains(bodyStr, `Precision="10"`) {
		t.Error("XML should contain Precision facet")
	}
	if !strings.Contains(bodyStr, `Scale="2"`) {
		t.Error("XML should contain Scale facet")
	}
	if !strings.Contains(bodyStr, `DefaultValue="AUTO"`) || !strings.Contains(bodyStr, `DefaultValue="pending"`) {
		t.Error("XML should contain DefaultValue facet")
	}

	// Verify time.Time is mapped to DateTimeOffset
	if !strings.Contains(bodyStr, `Type="Edm.DateTimeOffset"`) {
		t.Error("XML should map time.Time to Edm.DateTimeOffset")
	}

	// Verify navigation properties
	if !strings.Contains(bodyStr, `NavigationProperty Name="orders"`) {
		t.Error("XML should contain orders navigation property")
	}
	if !strings.Contains(bodyStr, `NavigationProperty Name="customer"`) {
		t.Error("XML should contain customer navigation property")
	}

	// Verify referential constraints
	if !strings.Contains(bodyStr, "<ReferentialConstraint>") {
		t.Error("XML should contain ReferentialConstraint elements")
	}
}

func TestMetadataIntegrationJSON(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	resp, err := http.Get(server.URL + "/$metadata?$format=json")
	if err != nil {
		t.Fatalf("Failed to get metadata: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status = %v, want %v", resp.StatusCode, http.StatusOK)
	}

	if contentType := resp.Header.Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Content-Type = %v, want application/json", contentType)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode JSON response: %v", err)
	}

	// Verify top-level structure
	if version, ok := response["$Version"].(string); !ok || version != "4.01" {
		t.Errorf("Expected $Version=4.01, got %v", response["$Version"])
	}

	odataService, ok := response["ODataService"].(map[string]interface{})
	if !ok {
		t.Fatal("ODataService not found in response")
	}

	// Verify Customer entity type
	customer, ok := odataService["Customer"].(map[string]interface{})
	if !ok {
		t.Fatal("Customer entity type not found")
	}

	// Check Customer properties with facets
	name, ok := customer["name"].(map[string]interface{})
	if !ok {
		t.Fatal("name property not found in Customer")
	}
	if maxLen, ok := name["$MaxLength"].(float64); !ok || maxLen != 100 {
		t.Errorf("Expected name MaxLength=100, got %v", name["$MaxLength"])
	}

	email, ok := customer["email"].(map[string]interface{})
	if !ok {
		t.Fatal("email property not found in Customer")
	}
	if nullable, ok := email["$Nullable"].(bool); !ok || !nullable {
		t.Errorf("Expected email to be nullable, got %v", email["$Nullable"])
	}

	// Check time.Time mapping
	createdAt, ok := customer["createdAt"].(map[string]interface{})
	if !ok {
		t.Fatal("createdAt property not found in Customer")
	}
	if edmType, ok := createdAt["$Type"].(string); !ok || edmType != "Edm.DateTimeOffset" {
		t.Errorf("Expected createdAt type=Edm.DateTimeOffset, got %v", createdAt["$Type"])
	}

	// Verify Order entity type with precision/scale
	order, ok := odataService["Order"].(map[string]interface{})
	if !ok {
		t.Fatal("Order entity type not found")
	}

	totalAmount, ok := order["totalAmount"].(map[string]interface{})
	if !ok {
		t.Fatal("totalAmount property not found in Order")
	}
	if precision, ok := totalAmount["$Precision"].(float64); !ok || precision != 10 {
		t.Errorf("Expected totalAmount Precision=10, got %v", totalAmount["$Precision"])
	}
	if scale, ok := totalAmount["$Scale"].(float64); !ok || scale != 2 {
		t.Errorf("Expected totalAmount Scale=2, got %v", totalAmount["$Scale"])
	}

	status, ok := order["status"].(map[string]interface{})
	if !ok {
		t.Fatal("status property not found in Order")
	}
	if defaultVal, ok := status["$DefaultValue"].(string); !ok || defaultVal != "pending" {
		t.Errorf("Expected status DefaultValue=pending, got %v", status["$DefaultValue"])
	}

	// Verify Product entity with default values
	product, ok := odataService["Product"].(map[string]interface{})
	if !ok {
		t.Fatal("Product entity type not found")
	}

	sku, ok := product["sku"].(map[string]interface{})
	if !ok {
		t.Fatal("sku property not found in Product")
	}
	if defaultVal, ok := sku["$DefaultValue"].(string); !ok || defaultVal != "AUTO" {
		t.Errorf("Expected sku DefaultValue=AUTO, got %v", sku["$DefaultValue"])
	}

	// Verify navigation properties with referential constraints
	customerNav, ok := order["customer"].(map[string]interface{})
	if !ok {
		t.Fatal("customer navigation property not found in Order")
	}
	if kind, ok := customerNav["$Kind"].(string); !ok || kind != "NavigationProperty" {
		t.Errorf("Expected $Kind=NavigationProperty, got %v", customerNav["$Kind"])
	}

	// Check referential constraints
	constraints, ok := customerNav["$ReferentialConstraint"].([]interface{})
	if !ok || len(constraints) == 0 {
		t.Error("$ReferentialConstraint not found or empty for customer navigation")
	}

	// Verify entity container
	container, ok := odataService["Container"].(map[string]interface{})
	if !ok {
		t.Fatal("Container not found")
	}

	// Verify entity sets
	if _, ok := container["Customers"]; !ok {
		t.Error("Customers entity set not found in container")
	}
	if _, ok := container["Orders"]; !ok {
		t.Error("Orders entity set not found in container")
	}
	if _, ok := container["Products"]; !ok {
		t.Error("Products entity set not found in container")
	}
}

func TestMetadataWithComplexRelationships(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	resp, err := http.Get(server.URL + "/$metadata?$format=json")
	if err != nil {
		t.Fatalf("Failed to get metadata: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode JSON response: %v", err)
	}

	odataService := response["ODataService"].(map[string]interface{})

	// Verify OrderItem has navigation to both Order and Product
	orderItem, ok := odataService["OrderItem"].(map[string]interface{})
	if !ok {
		t.Fatal("OrderItem entity type not found")
	}

	// Check order navigation
	orderNav, ok := orderItem["order"].(map[string]interface{})
	if !ok {
		t.Fatal("order navigation property not found in OrderItem")
	}
	if orderType, ok := orderNav["$Type"].(string); !ok || orderType != "ODataService.Order" {
		t.Errorf("Expected order type=ODataService.Order, got %v", orderNav["$Type"])
	}

	// Check product navigation
	productNav, ok := orderItem["product"].(map[string]interface{})
	if !ok {
		t.Fatal("product navigation property not found in OrderItem")
	}
	if productType, ok := productNav["$Type"].(string); !ok || productType != "ODataService.Product" {
		t.Errorf("Expected product type=ODataService.Product, got %v", productNav["$Type"])
	}

	// Verify collection navigation (one-to-many)
	customer, ok := odataService["Customer"].(map[string]interface{})
	if !ok {
		t.Fatal("Customer entity type not found")
	}

	ordersNav, ok := customer["orders"].(map[string]interface{})
	if !ok {
		t.Fatal("orders navigation property not found in Customer")
	}
	if collection, ok := ordersNav["$Collection"].(bool); !ok || !collection {
		t.Error("orders navigation should be a collection")
	}
}
