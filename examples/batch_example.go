package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Product represents a product entity
type Product struct {
	ID          uint    `json:"ID" gorm:"primaryKey" odata:"key"`
	Name        string  `json:"Name"`
	Price       float64 `json:"Price"`
	Category    string  `json:"Category"`
	Description string  `json:"Description"`
}

func main() {
	// Initialize database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// Auto-migrate
	if err := db.AutoMigrate(&Product{}); err != nil {
		log.Fatal(err)
	}

	// Create some sample data
	products := []Product{
		{ID: 1, Name: "Laptop", Price: 999.99, Category: "Electronics", Description: "High-performance laptop"},
		{ID: 2, Name: "Mouse", Price: 29.99, Category: "Electronics", Description: "Wireless mouse"},
		{ID: 3, Name: "Keyboard", Price: 79.99, Category: "Electronics", Description: "Mechanical keyboard"},
	}
	for _, p := range products {
		db.Create(&p)
	}

	// Initialize OData service
	service := odata.NewService(db)
	if err := service.RegisterEntity(&Product{}); err != nil {
		log.Fatal(err)
	}

	// Example 1: Multiple GET requests in a batch
	fmt.Println("=== Example 1: Multiple GET Requests ===")
	batchGetRequest := createBatchGetRequest()
	response := executeBatchRequest(service, batchGetRequest)
	fmt.Println(response)
	fmt.Println()

	// Example 2: Batch with changeset (atomic operations)
	fmt.Println("=== Example 2: Changeset (Atomic Operations) ===")
	batchChangesetRequest := createBatchChangesetRequest()
	response = executeBatchRequest(service, batchChangesetRequest)
	fmt.Println(response)
	fmt.Println()

	// Example 3: Mixed GET and changeset
	fmt.Println("=== Example 3: Mixed GET and Changeset ===")
	batchMixedRequest := createBatchMixedRequest()
	response = executeBatchRequest(service, batchMixedRequest)
	fmt.Println(response)
}

func createBatchGetRequest() string {
	boundary := "batch_36d5c8c6"
	return fmt.Sprintf(`--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /Products(1) HTTP/1.1
Host: localhost
Accept: application/json


--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /Products(2) HTTP/1.1
Host: localhost
Accept: application/json


--%s--
`, boundary, boundary, boundary)
}

func createBatchChangesetRequest() string {
	batchBoundary := "batch_36d5c8c6"
	changesetBoundary := "changeset_77162fcd"
	return fmt.Sprintf(`--%s
Content-Type: multipart/mixed; boundary=%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

POST /Products HTTP/1.1
Host: localhost
Content-Type: application/json

{"Name":"Tablet","Price":499.99,"Category":"Electronics","Description":"10-inch tablet"}

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

POST /Products HTTP/1.1
Host: localhost
Content-Type: application/json

{"Name":"Monitor","Price":299.99,"Category":"Electronics","Description":"27-inch monitor"}

--%s--

--%s--
`, batchBoundary, changesetBoundary, changesetBoundary, changesetBoundary, changesetBoundary, batchBoundary)
}

func createBatchMixedRequest() string {
	batchBoundary := "batch_36d5c8c6"
	changesetBoundary := "changeset_77162fcd"
	return fmt.Sprintf(`--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /Products(1) HTTP/1.1
Host: localhost
Accept: application/json


--%s
Content-Type: multipart/mixed; boundary=%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

PATCH /Products(1) HTTP/1.1
Host: localhost
Content-Type: application/json

{"Price":899.99}

--%s--

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /Products HTTP/1.1
Host: localhost
Accept: application/json


--%s--
`, batchBoundary, batchBoundary, changesetBoundary, changesetBoundary, changesetBoundary, batchBoundary, batchBoundary)
}

func executeBatchRequest(service *odata.Service, body string) string {
	req := httptest.NewRequest(http.MethodPost, "/$batch", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "multipart/mixed; boundary=batch_36d5c8c6")

	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	result := w.Result()
	defer result.Body.Close()

	responseBody, _ := io.ReadAll(result.Body)
	return fmt.Sprintf("Status: %d\n%s", result.StatusCode, string(responseBody))
}
