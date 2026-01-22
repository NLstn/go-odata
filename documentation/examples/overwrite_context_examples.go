//go:build example

// Package main demonstrates usage of key exported types in go-odata.
//
// This example shows how to use:
// 1. OverwriteContext with composite keys
// 2. QueryFilterProvider for authorization filters
// 3. SliceFilterFunc for custom filter evaluation
//
// Note: This is a standalone example file that demonstrates overwrite context concepts.
// It cannot be run directly with other example files due to package conflicts.
package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Example 1: OverwriteContext with Composite Keys
// ================================================

// OrderItem represents an entity with a composite key
type OrderItem struct {
	OrderID   int     `json:"orderId" odata:"key"`
	ProductID int     `json:"productId" odata:"key"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

// setupCompositeKeyExample demonstrates using OverwriteContext.EntityKeyValues
// to access individual key components in entities with composite keys.
func setupCompositeKeyExample(service *odata.Service) error {
	// Register the entity with composite key
	if err := service.RegisterVirtualEntity(&OrderItem{}); err != nil {
		return err
	}

	// Mock data store
	orderItems := map[string]*OrderItem{
		"1-101": {OrderID: 1, ProductID: 101, Quantity: 5, Price: 29.99},
		"1-102": {OrderID: 1, ProductID: 102, Quantity: 3, Price: 49.99},
		"2-101": {OrderID: 2, ProductID: 101, Quantity: 10, Price: 29.99},
	}

	// Set up overwrite handlers using EntityKeyValues for composite keys
	return service.SetEntityOverwrite("OrderItems", &odata.EntityOverwrite{
		GetEntity: func(ctx *odata.OverwriteContext) (interface{}, error) {
			// For composite keys, EntityKeyValues contains all key components
			// EntityKeyValues: map["OrderID": 1, "ProductID": 101]

			// Extract individual key values
			orderID, ok := ctx.EntityKeyValues["OrderID"]
			if !ok {
				return nil, fmt.Errorf("OrderID not found in key")
			}

			productID, ok := ctx.EntityKeyValues["ProductID"]
			if !ok {
				return nil, fmt.Errorf("ProductID not found in key")
			}

			// Convert to appropriate types
			var orderIDInt, productIDInt int
			switch v := orderID.(type) {
			case int:
				orderIDInt = v
			case int64:
				orderIDInt = int(v)
			case float64:
				orderIDInt = int(v)
			default:
				return nil, fmt.Errorf("unexpected type for OrderID: %T", orderID)
			}

			switch v := productID.(type) {
			case int:
				productIDInt = v
			case int64:
				productIDInt = int(v)
			case float64:
				productIDInt = int(v)
			default:
				return nil, fmt.Errorf("unexpected type for ProductID: %T", productID)
			}

			// Create composite key for lookup
			key := fmt.Sprintf("%d-%d", orderIDInt, productIDInt)

			// Retrieve the entity
			item, exists := orderItems[key]
			if !exists {
				return nil, &odata.ODataError{
					StatusCode: http.StatusNotFound,
					Code:       odata.ErrorCodeNotFound,
					Message:    fmt.Sprintf("OrderItem with OrderID=%d and ProductID=%d not found", orderIDInt, productIDInt),
				}
			}

			return item, nil
		},

		Update: func(ctx *odata.OverwriteContext, updateData map[string]interface{}, isFullReplace bool) (interface{}, error) {
			// Access composite key values with type checking
			orderIDRaw, ok := ctx.EntityKeyValues["OrderID"]
			if !ok {
				return nil, fmt.Errorf("OrderID not found in key")
			}
			productIDRaw, ok := ctx.EntityKeyValues["ProductID"]
			if !ok {
				return nil, fmt.Errorf("ProductID not found in key")
			}

			// Convert to appropriate types (handle various numeric types)
			var orderIDInt, productIDInt int
			switch v := orderIDRaw.(type) {
			case int:
				orderIDInt = v
			case int64:
				orderIDInt = int(v)
			case float64:
				orderIDInt = int(v)
			default:
				return nil, fmt.Errorf("unexpected type for OrderID: %T", orderIDRaw)
			}

			switch v := productIDRaw.(type) {
			case int:
				productIDInt = v
			case int64:
				productIDInt = int(v)
			case float64:
				productIDInt = int(v)
			default:
				return nil, fmt.Errorf("unexpected type for ProductID: %T", productIDRaw)
			}

			key := fmt.Sprintf("%d-%d", orderIDInt, productIDInt)
			item, exists := orderItems[key]
			if !exists {
				return nil, odata.ErrEntityNotFound
			}

			// Apply updates
			if quantity, ok := updateData["quantity"].(float64); ok {
				item.Quantity = int(quantity)
			}
			if price, ok := updateData["price"].(float64); ok {
				item.Price = price
			}

			return item, nil
		},

		Delete: func(ctx *odata.OverwriteContext) error {
			// Delete using composite key with type checking
			orderIDRaw, ok := ctx.EntityKeyValues["OrderID"]
			if !ok {
				return fmt.Errorf("OrderID not found in key")
			}
			productIDRaw, ok := ctx.EntityKeyValues["ProductID"]
			if !ok {
				return fmt.Errorf("ProductID not found in key")
			}

			// Convert to appropriate types
			var orderIDInt, productIDInt int
			switch v := orderIDRaw.(type) {
			case int:
				orderIDInt = v
			case int64:
				orderIDInt = int(v)
			case float64:
				orderIDInt = int(v)
			default:
				return fmt.Errorf("unexpected type for OrderID: %T", orderIDRaw)
			}

			switch v := productIDRaw.(type) {
			case int:
				productIDInt = v
			case int64:
				productIDInt = int(v)
			case float64:
				productIDInt = int(v)
			default:
				return fmt.Errorf("unexpected type for ProductID: %T", productIDRaw)
			}

			key := fmt.Sprintf("%d-%d", orderIDInt, productIDInt)
			if _, exists := orderItems[key]; !exists {
				return odata.ErrEntityNotFound
			}

			delete(orderItems, key)
			return nil
		},
	})
}

// Example 2: QueryFilterProvider for Row-Level Security
// ======================================================

// Product represents a product entity
type Product struct {
	ID       int     `json:"id" gorm:"primarykey" odata:"key"`
	Name     string  `json:"name"`
	Price    float64 `json:"price"`
	TenantID string  `json:"tenantId"` // For multi-tenant filtering
	IsActive bool    `json:"isActive"`
}

// TenantFilterPolicy implements QueryFilterProvider to automatically
// filter queries based on the user's tenant.
type TenantFilterPolicy struct{}

func (p *TenantFilterPolicy) Authorize(ctx odata.AuthContext, resource odata.ResourceDescriptor, op odata.Operation) odata.Decision {
	// Allow metadata operations
	if op == odata.OperationMetadata {
		return odata.Allow()
	}

	// Verify user has a tenant
	if _, ok := ctx.Claims["tenant_id"]; !ok {
		return odata.Deny("User must be associated with a tenant")
	}

	return odata.Allow()
}

// QueryFilter adds automatic tenant filtering to all queries.
// This ensures users can only see data from their own tenant.
func (p *TenantFilterPolicy) QueryFilter(ctx odata.AuthContext, resource odata.ResourceDescriptor, op odata.Operation) (*odata.FilterExpression, error) {
	// Skip filtering for metadata operations
	if op == odata.OperationMetadata {
		return nil, nil
	}

	// Extract tenant ID from auth context
	tenantID, ok := ctx.Claims["tenant_id"]
	if !ok {
		return nil, fmt.Errorf("tenant_id not found in claims")
	}

	// Return a filter expression: TenantID eq 'user-tenant-id'
	// This filter is automatically combined with any user-provided $filter
	filter := &odata.FilterExpression{
		Property: "TenantID",
		Operator: odata.OpEqual,
		Value:    tenantID,
	}

	return filter, nil
}

// Example 3: SliceFilterFunc for Custom Filter Evaluation
// ========================================================

// createProductFilterFunc creates a custom filter evaluator for Product entities.
// This allows applying OData filters to in-memory slices of products.
func createProductFilterFunc() odata.SliceFilterFunc[Product] {
	return func(item Product, filter *odata.FilterExpression) (bool, error) {
		if filter == nil {
			return true, nil
		}

		// Handle logical operators (AND, OR)
		if filter.Logical != "" {
			if filter.Left == nil || filter.Right == nil {
				return false, fmt.Errorf("logical operator requires left and right expressions")
			}

			leftMatch, err := createProductFilterFunc()(item, filter.Left)
			if err != nil {
				return false, err
			}

			rightMatch, err := createProductFilterFunc()(item, filter.Right)
			if err != nil {
				return false, err
			}

			if filter.Logical == odata.LogicalAnd {
				return leftMatch && rightMatch, nil
			}
			if filter.Logical == odata.LogicalOr {
				return leftMatch || rightMatch, nil
			}

			return false, fmt.Errorf("unsupported logical operator: %s", filter.Logical)
		}

		// Handle property filters
		switch filter.Property {
		case "Name", "name":
			return evaluateStringFilter(item.Name, filter)

		case "Price", "price":
			return evaluateNumericFilter(item.Price, filter)

		case "IsActive", "isActive":
			return evaluateBoolFilter(item.IsActive, filter)

		case "TenantID", "tenantId":
			return evaluateStringFilter(item.TenantID, filter)

		default:
			// Unknown property - return false or error based on requirements
			return false, fmt.Errorf("unknown property in filter: %s", filter.Property)
		}
	}
}

// Helper function to evaluate string filters
func evaluateStringFilter(value string, filter *odata.FilterExpression) (bool, error) {
	switch filter.Operator {
	case odata.OpEqual:
		if filterValue, ok := filter.Value.(string); ok {
			return value == filterValue, nil
		}
	case odata.OpNotEqual:
		if filterValue, ok := filter.Value.(string); ok {
			return value != filterValue, nil
		}
	case odata.OpContains:
		if filterValue, ok := filter.Value.(string); ok {
			return strings.Contains(strings.ToLower(value), strings.ToLower(filterValue)), nil
		}
	case odata.OpStartsWith:
		if filterValue, ok := filter.Value.(string); ok {
			return strings.HasPrefix(strings.ToLower(value), strings.ToLower(filterValue)), nil
		}
	case odata.OpEndsWith:
		if filterValue, ok := filter.Value.(string); ok {
			return strings.HasSuffix(strings.ToLower(value), strings.ToLower(filterValue)), nil
		}
	}
	return false, fmt.Errorf("unsupported string operator: %s", filter.Operator)
}

// Helper function to evaluate numeric filters
func evaluateNumericFilter(value float64, filter *odata.FilterExpression) (bool, error) {
	var filterValue float64
	switch v := filter.Value.(type) {
	case float64:
		filterValue = v
	case int:
		filterValue = float64(v)
	case int64:
		filterValue = float64(v)
	case string:
		var err error
		filterValue, err = strconv.ParseFloat(v, 64)
		if err != nil {
			return false, fmt.Errorf("invalid numeric value: %s", v)
		}
	default:
		return false, fmt.Errorf("unsupported numeric type: %T", filter.Value)
	}

	switch filter.Operator {
	case odata.OpEqual:
		return value == filterValue, nil
	case odata.OpNotEqual:
		return value != filterValue, nil
	case odata.OpGreaterThan:
		return value > filterValue, nil
	case odata.OpGreaterThanOrEqual:
		return value >= filterValue, nil
	case odata.OpLessThan:
		return value < filterValue, nil
	case odata.OpLessThanOrEqual:
		return value <= filterValue, nil
	}
	return false, fmt.Errorf("unsupported numeric operator: %s", filter.Operator)
}

// Helper function to evaluate boolean filters
func evaluateBoolFilter(value bool, filter *odata.FilterExpression) (bool, error) {
	if filter.Operator != odata.OpEqual && filter.Operator != odata.OpNotEqual {
		return false, fmt.Errorf("unsupported boolean operator: %s", filter.Operator)
	}

	var filterValue bool
	switch v := filter.Value.(type) {
	case bool:
		filterValue = v
	case string:
		filterValue = v == "true"
	default:
		return false, fmt.Errorf("unsupported boolean type: %T", filter.Value)
	}

	if filter.Operator == odata.OpEqual {
		return value == filterValue, nil
	}
	return value != filterValue, nil
}

// Example 4: Using SliceFilterFunc with ApplyQueryOptionsToSlice
// ===============================================================

func setupVirtualProductsWithSliceFilter(service *odata.Service) error {
	// Register virtual entity
	if err := service.RegisterVirtualEntity(&Product{}); err != nil {
		return err
	}

	// Mock product data
	products := []Product{
		{ID: 1, Name: "Laptop", Price: 999.99, TenantID: "tenant-a", IsActive: true},
		{ID: 2, Name: "Mouse", Price: 29.99, TenantID: "tenant-a", IsActive: true},
		{ID: 3, Name: "Keyboard", Price: 79.99, TenantID: "tenant-b", IsActive: true},
		{ID: 4, Name: "Monitor", Price: 299.99, TenantID: "tenant-a", IsActive: false},
	}

	// Set up GetCollection handler with custom filter
	return service.SetEntityOverwrite("Products", &odata.EntityOverwrite{
		GetCollection: func(ctx *odata.OverwriteContext) (*odata.CollectionResult, error) {
			// Apply OData query options to in-memory slice
			filtered, err := odata.ApplyQueryOptionsToSlice(
				products,
				ctx.QueryOptions,
				createProductFilterFunc(), // Use our custom filter function
			)
			if err != nil {
				return nil, err
			}

			// Calculate count if requested
			var count *int64
			if ctx.QueryOptions != nil && ctx.QueryOptions.Count {
				c := int64(len(filtered))
				count = &c
			}

			return &odata.CollectionResult{
				Items: filtered,
				Count: count,
			}, nil
		},
	})
}

// Example 5: Complete Application
// ================================

func main() {
	// Initialize database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// Create OData service
	service, err := odata.NewService(db)
	if err != nil {
		log.Fatal(err)
	}

	// Example 1: Set up composite key entity
	if err := setupCompositeKeyExample(service); err != nil {
		log.Fatalf("Failed to set up composite key example: %v", err)
	}

	// Example 2 & 4: Set up virtual products with custom filter
	if err := setupVirtualProductsWithSliceFilter(service); err != nil {
		log.Fatalf("Failed to set up products: %v", err)
	}

	// Example 3: Set up authorization policy with query filter
	policy := &TenantFilterPolicy{}
	if err := service.SetPolicy(policy); err != nil {
		log.Fatal(err)
	}

	// Start server
	log.Println("OData service running on :8080")
	log.Println("Examples:")
	log.Println("  - Composite keys: http://localhost:8080/OrderItems(OrderID=1,ProductID=101)")
	log.Println("  - Virtual products: http://localhost:8080/Products?$filter=Price gt 50")
	log.Fatal(http.ListenAndServe(":8080", service))
}
