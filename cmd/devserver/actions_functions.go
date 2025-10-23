package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	"github.com/NLstn/go-odata/devserver/entities"
	"github.com/nlstn/go-odata"
	"gorm.io/gorm"
)

// registerFunctions registers example OData functions
func registerFunctions(service *odata.Service, db *gorm.DB) {
	// Unbound function: GetTopProducts - returns top N most expensive products
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
		fmt.Printf("Failed to register GetTopProducts function: %v\n", err)
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

			// Extract product ID from URL
			// For simplicity, we'll parse it from the request URL
			// In a real implementation, ctx would contain the bound entity
			path := r.URL.Path
			var productID uint
			if _, err := fmt.Sscanf(path, "/Products(%d)/GetTotalPrice", &productID); err != nil {
				return nil, fmt.Errorf("invalid product ID")
			}

			var product entities.Product
			if err := db.First(&product, productID).Error; err != nil {
				return nil, err
			}

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
}

// registerActions registers example OData actions
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

			// Extract product ID from URL
			path := r.URL.Path
			var productID uint
			if _, err := fmt.Sscanf(path, "/Products(%d)/ApplyDiscount", &productID); err != nil {
				return fmt.Errorf("invalid product ID")
			}

			var product entities.Product
			if err := db.First(&product, productID).Error; err != nil {
				return err
			}

			// Apply discount
			product.Price = product.Price * (1 - percentage/100)
			if err := db.Save(&product).Error; err != nil {
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
			priceMap := make(map[uint]float64)
			for _, p := range sampleProducts {
				priceMap[p.ID] = p.Price
			}

			var products []entities.Product
			if err := db.Find(&products).Error; err != nil {
				return err
			}

			updatedCount := 0
			for i := range products {
				if originalPrice, exists := priceMap[products[i].ID]; exists {
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

			// Extract product ID from URL
			path := r.URL.Path
			var productID uint
			if _, err := fmt.Sscanf(path, "/Products(%d)/IncreasePrice", &productID); err != nil {
				return fmt.Errorf("invalid product ID")
			}

			var product entities.Product
			if err := db.First(&product, productID).Error; err != nil {
				return err
			}

			// Increase price
			product.Price += amount
			if err := db.Save(&product).Error; err != nil {
				return err
			}

			// Return 204 No Content for void actions
			w.WriteHeader(http.StatusNoContent)
			return nil
		},
	}); err != nil {
		fmt.Printf("Failed to register IncreasePrice action: %v\n", err)
	}
}
