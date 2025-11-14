package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	"github.com/nlstn/go-odata"
	"github.com/nlstn/go-odata/cmd/complianceserver/entities"
	"gorm.io/gorm"
)

// registerFunctions registers OData functions for compliance testing
func registerFunctions(service *odata.Service, db *gorm.DB) {
	// Unbound function: GetTopProducts - overload with no parameters
	if err := service.RegisterFunction(odata.FunctionDefinition{
		Name:       "GetTopProducts",
		IsBound:    false,
		Parameters: []odata.ParameterDefinition{},
		ReturnType: reflect.TypeOf([]entities.Product{}),
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) (interface{}, error) {
			var products []entities.Product
			if err := db.Order("price DESC").Limit(10).Find(&products).Error; err != nil {
				return nil, err
			}
			return products, nil
		},
	}); err != nil {
		fmt.Printf("Failed to register GetTopProducts (no params) function: %v\n", err)
	}

	// Unbound function: GetTopProducts - overload with count parameter
	if err := service.RegisterFunction(odata.FunctionDefinition{
		Name:    "GetTopProducts",
		IsBound: false,
		Parameters: []odata.ParameterDefinition{
			{Name: "count", Type: reflect.TypeOf(int64(0)), Required: true},
		},
		ReturnType: reflect.TypeOf([]entities.Product{}),
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) (interface{}, error) {
			count := params["count"].(int64)
			var products []entities.Product
			if err := db.Order("price DESC").Limit(int(count)).Find(&products).Error; err != nil {
				return nil, err
			}
			return products, nil
		},
	}); err != nil {
		fmt.Printf("Failed to register GetTopProducts (with count) function: %v\n", err)
	}

	// Unbound function: GetTopProducts - overload with count and category parameters
	if err := service.RegisterFunction(odata.FunctionDefinition{
		Name:    "GetTopProducts",
		IsBound: false,
		Parameters: []odata.ParameterDefinition{
			{Name: "count", Type: reflect.TypeOf(int64(0)), Required: true},
			{Name: "category", Type: reflect.TypeOf(""), Required: true},
		},
		ReturnType: reflect.TypeOf([]entities.Product{}),
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) (interface{}, error) {
			count := params["count"].(int64)
			category := params["category"].(string)

			var products []entities.Product
			if err := db.Joins("JOIN categories ON categories.id = products.category_id").
				Where("categories.name = ?", category).
				Order("products.price DESC").
				Limit(int(count)).
				Find(&products).Error; err != nil {
				return nil, err
			}
			return products, nil
		},
	}); err != nil {
		fmt.Printf("Failed to register GetTopProducts (with count and category) function: %v\n", err)
	}

	// Bound function: GetTotalPrice - calculates total price with tax
	if err := service.RegisterFunction(odata.FunctionDefinition{
		Name:      "GetTotalPrice",
		IsBound:   true,
		EntitySet: "Products",
		Parameters: []odata.ParameterDefinition{
			{Name: "taxRate", Type: reflect.TypeOf(float64(0)), Required: true},
		},
		ReturnType: reflect.TypeOf(float64(0)),
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) (interface{}, error) {
			taxRate := params["taxRate"].(float64)

			// Use the bound context which contains the product entity
			product := ctx.(*entities.Product)

			totalPrice := product.Price * (1 + taxRate)
			return totalPrice, nil
		},
	}); err != nil {
		fmt.Printf("Failed to register GetTotalPrice function: %v\n", err)
	}

	// Unbound function: GetProductStats - returns statistics about products
	if err := service.RegisterFunction(odata.FunctionDefinition{
		Name:       "GetProductStats",
		IsBound:    false,
		Parameters: []odata.ParameterDefinition{},
		ReturnType: reflect.TypeOf(map[string]interface{}{}),
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) (interface{}, error) {
			var count int64
			var avgPrice float64
			var maxPrice float64
			var minPrice float64

			db.Model(&entities.Product{}).Count(&count)
			db.Model(&entities.Product{}).Select("AVG(price)").Row().Scan(&avgPrice)
			db.Model(&entities.Product{}).Select("MAX(price)").Row().Scan(&maxPrice)
			db.Model(&entities.Product{}).Select("MIN(price)").Row().Scan(&minPrice)

			return map[string]interface{}{
				"totalProducts": count,
				"averagePrice":  avgPrice,
				"maxPrice":      maxPrice,
				"minPrice":      minPrice,
			}, nil
		},
	}); err != nil {
		fmt.Printf("Failed to register GetProductStats function: %v\n", err)
	}

	// Bound function: GetRelatedProducts - returns products in the same category
	if err := service.RegisterFunction(odata.FunctionDefinition{
		Name:       "GetRelatedProducts",
		IsBound:    true,
		EntitySet:  "Products",
		Parameters: []odata.ParameterDefinition{},
		ReturnType: reflect.TypeOf([]entities.Product{}),
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) (interface{}, error) {
			product := ctx.(*entities.Product)

			var relatedProducts []entities.Product
			if err := db.Where("category_id = ? AND id != ?", product.CategoryID, product.ID).
				Limit(5).Find(&relatedProducts).Error; err != nil {
				return nil, err
			}

			return relatedProducts, nil
		},
	}); err != nil {
		fmt.Printf("Failed to register GetRelatedProducts function: %v\n", err)
	}

	// Unbound function: FindProducts - searches products by name and max price
	if err := service.RegisterFunction(odata.FunctionDefinition{
		Name:    "FindProducts",
		IsBound: false,
		Parameters: []odata.ParameterDefinition{
			{Name: "name", Type: reflect.TypeOf(""), Required: true},
			{Name: "maxPrice", Type: reflect.TypeOf(float64(0)), Required: true},
		},
		ReturnType: reflect.TypeOf([]entities.Product{}),
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) (interface{}, error) {
			name := params["name"].(string)
			maxPrice := params["maxPrice"].(float64)

			var products []entities.Product
			if err := db.Where("name LIKE ? AND price <= ?", "%"+name+"%", maxPrice).
				Find(&products).Error; err != nil {
				return nil, err
			}

			return products, nil
		},
	}); err != nil {
		fmt.Printf("Failed to register FindProducts function: %v\n", err)
	}

	// Bound function on collection: GetAveragePrice - calculates average price of products
	if err := service.RegisterFunction(odata.FunctionDefinition{
		Name:       "GetAveragePrice",
		IsBound:    true,
		EntitySet:  "Products",
		Parameters: []odata.ParameterDefinition{},
		ReturnType: reflect.TypeOf(float64(0)),
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) (interface{}, error) {
			var avgPrice float64
			if err := db.Model(&entities.Product{}).Select("AVG(price)").Row().Scan(&avgPrice); err != nil {
				return nil, err
			}
			return avgPrice, nil
		},
	}); err != nil {
		fmt.Printf("Failed to register GetAveragePrice function: %v\n", err)
	}

	// Overloaded function: Calculate - with one parameter
	if err := service.RegisterFunction(odata.FunctionDefinition{
		Name:    "Calculate",
		IsBound: false,
		Parameters: []odata.ParameterDefinition{
			{Name: "value", Type: reflect.TypeOf(int64(0)), Required: true},
		},
		ReturnType: reflect.TypeOf(int64(0)),
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) (interface{}, error) {
			value := params["value"].(int64)
			return value * 2, nil
		},
	}); err != nil {
		fmt.Printf("Failed to register Calculate (one param) function: %v\n", err)
	}

	// Overloaded function: Calculate - with two parameters
	if err := service.RegisterFunction(odata.FunctionDefinition{
		Name:    "Calculate",
		IsBound: false,
		Parameters: []odata.ParameterDefinition{
			{Name: "a", Type: reflect.TypeOf(int64(0)), Required: true},
			{Name: "b", Type: reflect.TypeOf(int64(0)), Required: true},
		},
		ReturnType: reflect.TypeOf(int64(0)),
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) (interface{}, error) {
			a := params["a"].(int64)
			b := params["b"].(int64)
			return a + b, nil
		},
	}); err != nil {
		fmt.Printf("Failed to register Calculate (two params) function: %v\n", err)
	}

	// Overloaded function: Convert - with string parameter
	if err := service.RegisterFunction(odata.FunctionDefinition{
		Name:    "Convert",
		IsBound: false,
		Parameters: []odata.ParameterDefinition{
			{Name: "input", Type: reflect.TypeOf(""), Required: true},
		},
		ReturnType: reflect.TypeOf(""),
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) (interface{}, error) {
			input := params["input"].(string)
			return "String: " + input, nil
		},
	}); err != nil {
		fmt.Printf("Failed to register Convert (string) function: %v\n", err)
	}

	// Overloaded function: Convert - with number parameter
	if err := service.RegisterFunction(odata.FunctionDefinition{
		Name:    "Convert",
		IsBound: false,
		Parameters: []odata.ParameterDefinition{
			{Name: "number", Type: reflect.TypeOf(int64(0)), Required: true},
		},
		ReturnType: reflect.TypeOf(""),
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) (interface{}, error) {
			number := params["number"].(int64)
			return fmt.Sprintf("Number: %d", number), nil
		},
	}); err != nil {
		fmt.Printf("Failed to register Convert (number) function: %v\n", err)
	}

	// Bound function: CalculatePrice - with discount parameter
	if err := service.RegisterFunction(odata.FunctionDefinition{
		Name:      "CalculatePrice",
		IsBound:   true,
		EntitySet: "Products",
		Parameters: []odata.ParameterDefinition{
			{Name: "discount", Type: reflect.TypeOf(float64(0)), Required: true},
		},
		ReturnType: reflect.TypeOf(float64(0)),
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) (interface{}, error) {
			product := ctx.(*entities.Product)
			discount := params["discount"].(float64)
			return product.Price * (1.0 - discount/100.0), nil
		},
	}); err != nil {
		fmt.Printf("Failed to register CalculatePrice (one param) function: %v\n", err)
	}

	// Bound function: CalculatePrice - with discount and tax parameters
	if err := service.RegisterFunction(odata.FunctionDefinition{
		Name:      "CalculatePrice",
		IsBound:   true,
		EntitySet: "Products",
		Parameters: []odata.ParameterDefinition{
			{Name: "discount", Type: reflect.TypeOf(float64(0)), Required: true},
			{Name: "tax", Type: reflect.TypeOf(float64(0)), Required: true},
		},
		ReturnType: reflect.TypeOf(float64(0)),
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) (interface{}, error) {
			product := ctx.(*entities.Product)
			discount := params["discount"].(float64)
			tax := params["tax"].(float64)
			discountedPrice := product.Price * (1.0 - discount/100.0)
			return discountedPrice * (1.0 + tax/100.0), nil
		},
	}); err != nil {
		fmt.Printf("Failed to register CalculatePrice (two params) function: %v\n", err)
	}

	// Bound function: GetInfo for Products
	if err := service.RegisterFunction(odata.FunctionDefinition{
		Name:      "GetInfo",
		IsBound:   true,
		EntitySet: "Products",
		Parameters: []odata.ParameterDefinition{
			{Name: "format", Type: reflect.TypeOf(""), Required: true},
		},
		ReturnType: reflect.TypeOf(""),
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) (interface{}, error) {
			product := ctx.(*entities.Product)
			format := params["format"].(string)
			return fmt.Sprintf("Product: %s (Price: %.2f) in %s format", product.Name, product.Price, format), nil
		},
	}); err != nil {
		fmt.Printf("Failed to register GetInfo (Products) function: %v\n", err)
	}
}

// registerActions registers OData actions for compliance testing
func registerActions(service *odata.Service, db *gorm.DB) {
	// Bound action: ApplyDiscount - applies a discount to a product
	if err := service.RegisterAction(odata.ActionDefinition{
		Name:      "ApplyDiscount",
		IsBound:   true,
		EntitySet: "Products",
		Parameters: []odata.ParameterDefinition{
			{Name: "percentage", Type: reflect.TypeOf(float64(0)), Required: true},
		},
		ReturnType: reflect.TypeOf(entities.Product{}),
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) error {
			percentage := params["percentage"].(float64)
			product := ctx.(*entities.Product)

			// Apply discount
			product.Price = product.Price * (1 - percentage/100)
			if err := db.Save(product).Error; err != nil {
				return err
			}

			// Return updated product
			w.Header().Set("Content-Type", "application/json;odata.metadata=minimal")
			w.WriteHeader(http.StatusOK)

			response := map[string]interface{}{
				"@odata.context": "$metadata#Products/$entity",
				"value":          product,
			}

			return json.NewEncoder(w).Encode(response)
		},
	}); err != nil {
		fmt.Printf("Failed to register ApplyDiscount action: %v\n", err)
	}

	// Unbound action: ResetAllPrices - resets all product prices to default values
	if err := service.RegisterAction(odata.ActionDefinition{
		Name:       "ResetAllPrices",
		IsBound:    false,
		Parameters: []odata.ParameterDefinition{},
		ReturnType: reflect.TypeOf(map[string]interface{}{}),
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) error {
			// Get all products and reset their prices to original sample values
			sampleProducts := entities.GetSampleProducts()
			priceMap := make(map[string]float64)
			for _, p := range sampleProducts {
				if p.ID.String() != "" {
					priceMap[p.ID.String()] = p.Price
				}
			}

			var products []entities.Product
			if err := db.Find(&products).Error; err != nil {
				return err
			}

			updatedCount := 0
			for i := range products {
				if originalPrice, exists := priceMap[products[i].ID.String()]; exists {
					products[i].Price = originalPrice
					if err := db.Save(&products[i]).Error; err != nil {
						return err
					}
					updatedCount++
				}
			}

			// Return result
			w.Header().Set("Content-Type", "application/json;odata.metadata=minimal")
			w.WriteHeader(http.StatusOK)

			response := map[string]interface{}{
				"@odata.context": "$metadata#Edm.String",
				"value": map[string]interface{}{
					"message":      "Prices reset successfully",
					"updatedCount": updatedCount,
				},
			}

			return json.NewEncoder(w).Encode(response)
		},
	}); err != nil {
		fmt.Printf("Failed to register ResetAllPrices action: %v\n", err)
	}

	// Bound action: IncreasePrice - increases a product's price by a fixed amount
	if err := service.RegisterAction(odata.ActionDefinition{
		Name:      "IncreasePrice",
		IsBound:   true,
		EntitySet: "Products",
		Parameters: []odata.ParameterDefinition{
			{Name: "amount", Type: reflect.TypeOf(float64(0)), Required: true},
		},
		ReturnType: nil, // No return value
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) error {
			amount := params["amount"].(float64)
			product := ctx.(*entities.Product)

			// Increase price
			product.Price += amount
			if err := db.Save(product).Error; err != nil {
				return err
			}

			// Return 204 No Content for void actions
			w.WriteHeader(http.StatusNoContent)
			return nil
		},
	}); err != nil {
		fmt.Printf("Failed to register IncreasePrice action: %v\n", err)
	}

	// Unbound action: ResetProducts - resets products data
	if err := service.RegisterAction(odata.ActionDefinition{
		Name:       "ResetProducts",
		IsBound:    false,
		Parameters: []odata.ParameterDefinition{},
		ReturnType: nil, // No return value
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) error {
			// Reset all products to original sample values
			sampleProducts := entities.GetSampleProducts()
			for _, p := range sampleProducts {
				db.Save(&p)
			}

			// Return 204 No Content
			w.WriteHeader(http.StatusNoContent)
			return nil
		},
	}); err != nil {
		fmt.Printf("Failed to register ResetProducts action: %v\n", err)
	}

	// Bound action: Activate - activates a product
	if err := service.RegisterAction(odata.ActionDefinition{
		Name:       "Activate",
		IsBound:    true,
		EntitySet:  "Products",
		Parameters: []odata.ParameterDefinition{},
		ReturnType: nil, // No return value
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) error {
			product := ctx.(*entities.Product)

			// Remove discontinued flag from status (activate the product)
			product.Status = product.Status &^ entities.ProductStatusDiscontinued
			// Add in stock flag if not present
			product.Status = product.Status | entities.ProductStatusInStock

			if err := db.Save(product).Error; err != nil {
				return err
			}

			// Return 204 No Content
			w.WriteHeader(http.StatusNoContent)
			return nil
		},
	}); err != nil {
		fmt.Printf("Failed to register Activate action: %v\n", err)
	}

	// Bound action: CalculateDiscount - calculates discount for a product
	if err := service.RegisterAction(odata.ActionDefinition{
		Name:      "CalculateDiscount",
		IsBound:   true,
		EntitySet: "Products",
		Parameters: []odata.ParameterDefinition{
			{Name: "percentage", Type: reflect.TypeOf(float64(0)), Required: true},
		},
		ReturnType: reflect.TypeOf(float64(0)),
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) error {
			percentage := params["percentage"].(float64)
			product := ctx.(*entities.Product)

			// Calculate discounted price (but don't save it)
			discountedPrice := product.Price * (1 - percentage/100)

			// Return the calculated value
			w.Header().Set("Content-Type", "application/json;odata.metadata=minimal")
			w.WriteHeader(http.StatusOK)

			response := map[string]interface{}{
				"@odata.context": "$metadata#Edm.Double",
				"value":          discountedPrice,
			}

			return json.NewEncoder(w).Encode(response)
		},
	}); err != nil {
		fmt.Printf("Failed to register CalculateDiscount action: %v\n", err)
	}

	// Bound action on collection: MarkAllAsReviewed - marks all products as reviewed
	if err := service.RegisterAction(odata.ActionDefinition{
		Name:       "MarkAllAsReviewed",
		IsBound:    true,
		EntitySet:  "Products",
		Parameters: []odata.ParameterDefinition{},
		ReturnType: nil, // No return value
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) error {
			// Update all products - set a hypothetical "reviewed" flag
			// Since Product entity doesn't have this field, we'll just return success
			// In a real scenario, this would update a field on all products

			// Return 204 No Content
			w.WriteHeader(http.StatusNoContent)
			return nil
		},
	}); err != nil {
		fmt.Printf("Failed to register MarkAllAsReviewed action: %v\n", err)
	}

	// Overloaded action: Process - with percentage parameter only
	if err := service.RegisterAction(odata.ActionDefinition{
		Name:    "Process",
		IsBound: false,
		Parameters: []odata.ParameterDefinition{
			{Name: "percentage", Type: reflect.TypeOf(float64(0)), Required: true},
		},
		ReturnType: nil,
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) error {
			percentage := params["percentage"].(float64)
			multiplier := 1.0 - (percentage / 100.0)

			if err := db.Model(&entities.Product{}).Where("1 = 1").
				Update("price", gorm.Expr("price * ?", multiplier)).Error; err != nil {
				return err
			}

			w.WriteHeader(http.StatusNoContent)
			return nil
		},
	}); err != nil {
		fmt.Printf("Failed to register Process (one param) action: %v\n", err)
	}

	// Overloaded action: Process - with percentage and minPrice parameters
	if err := service.RegisterAction(odata.ActionDefinition{
		Name:    "Process",
		IsBound: false,
		Parameters: []odata.ParameterDefinition{
			{Name: "percentage", Type: reflect.TypeOf(float64(0)), Required: true},
			{Name: "minPrice", Type: reflect.TypeOf(float64(0)), Required: true},
		},
		ReturnType: nil,
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) error {
			percentage := params["percentage"].(float64)
			minPrice := params["minPrice"].(float64)
			multiplier := 1.0 - (percentage / 100.0)

			// Apply discount to products with price >= minPrice
			if err := db.Model(&entities.Product{}).
				Where("price >= ?", minPrice).
				Update("price", gorm.Expr("price * ?", multiplier)).Error; err != nil {
				return err
			}

			w.WriteHeader(http.StatusNoContent)
			return nil
		},
	}); err != nil {
		fmt.Printf("Failed to register Process (two params) action: %v\n", err)
	}
}
