package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/NLstn/go-odata/perfserver/entities"
	"github.com/nlstn/go-odata"
	"gorm.io/gorm"
)

// seedDatabase initializes the database with extensive sample data for performance testing
// This function drops and recreates all tables to ensure a clean state
func seedDatabase(db *gorm.DB, extensive bool) error {
	// Get the database dialect name to determine the database type
	dialectName := db.Name()

	// Drop all tables (GORM handles the correct order based on foreign keys)
	if err := db.Migrator().DropTable(&entities.ProductDescription{}, &entities.Product{}, &entities.Category{}, &entities.CompanyInfo{}, &entities.APIKey{}); err != nil {
		return fmt.Errorf("failed to drop tables: %w", err)
	}

	// For PostgreSQL, explicitly reset sequences after dropping tables
	// This ensures auto-increment columns start from 1 after reseeding
	if dialectName == "postgres" {
		// List of table names that need sequence resets (tables with auto-increment primary keys)
		tables := []string{"categories", "products", "product_descriptions", "company_infos", "api_keys"}
		for _, table := range tables {
			// PostgreSQL convention: sequence name is table_column_seq
			// GORM uses lowercase table names and "id" for primary key columns by default
			sequenceName := table + "_id_seq"
			if err := db.Exec(fmt.Sprintf("DROP SEQUENCE IF EXISTS %s CASCADE", sequenceName)).Error; err != nil {
				// Log but don't fail - sequence might not exist
				fmt.Printf("Note: Could not drop sequence %s: %v\n", sequenceName, err)
			}
		}
	}

	// Recreate tables with fresh schema (auto-increment counters are automatically reset)
	if err := db.AutoMigrate(&entities.Category{}, &entities.Product{}, &entities.ProductDescription{}, &entities.CompanyInfo{}, &entities.APIKey{}); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	// Seed categories first (products reference categories)
	var sampleCategories []entities.Category
	if extensive {
		sampleCategories = generateExtensiveCategories()
	} else {
		sampleCategories = entities.GetSampleCategories()
	}
	if err := db.Create(&sampleCategories).Error; err != nil {
		return fmt.Errorf("failed to seed categories: %w", err)
	}

	// Seed products in batches to avoid SQL variable limits
	var sampleProducts []entities.Product
	if extensive {
		sampleProducts = generateExtensiveProducts(sampleCategories)
	} else {
		sampleProducts = entities.GetSampleProducts()
	}

	// Insert products in batches of 100
	batchSize := 100
	for i := 0; i < len(sampleProducts); i += batchSize {
		end := i + batchSize
		if end > len(sampleProducts) {
			end = len(sampleProducts)
		}
		batch := sampleProducts[i:end]
		if err := db.Create(&batch).Error; err != nil {
			return fmt.Errorf("failed to seed products batch %d-%d: %w", i, end, err)
		}
	}

	// Seed product descriptions in batches to avoid SQL variable limits
	var sampleDescriptions []entities.ProductDescription
	if extensive {
		sampleDescriptions = generateExtensiveProductDescriptions(sampleProducts)
	} else {
		sampleDescriptions = entities.GetSampleProductDescriptions()
	}

	// Insert descriptions in batches of 100
	for i := 0; i < len(sampleDescriptions); i += batchSize {
		end := i + batchSize
		if end > len(sampleDescriptions) {
			end = len(sampleDescriptions)
		}
		batch := sampleDescriptions[i:end]
		if err := db.Create(&batch).Error; err != nil {
			return fmt.Errorf("failed to seed product descriptions batch %d-%d: %w", i, end, err)
		}
	}

	// Seed company info singleton
	companyInfo := entities.GetCompanyInfo()
	if err := db.Create(&companyInfo).Error; err != nil {
		return fmt.Errorf("failed to seed company info: %w", err)
	}

	var apiKeys []entities.APIKey
	if extensive {
		apiKeys = entities.GenerateExtensiveAPIKeys(1000)
	} else {
		apiKeys = entities.GetSampleAPIKeys()
	}
	if err := db.Create(&apiKeys).Error; err != nil {
		return fmt.Errorf("failed to seed API keys: %w", err)
	}

	fmt.Printf("Database seeded with %d categories, %d products, %d descriptions, %d API keys, and company info\n",
		len(sampleCategories), len(sampleProducts), len(sampleDescriptions), len(apiKeys))
	return nil
}

// generateExtensiveCategories creates a large number of categories for performance testing
func generateExtensiveCategories() []entities.Category {
	categories := make([]entities.Category, 0, 100)
	categoryNames := []string{
		"Electronics", "Computers", "Accessories", "Networking", "Gaming",
		"Kitchen", "Home", "Garden", "Furniture", "Décor",
		"Sports", "Fitness", "Outdoor", "Camping", "Cycling",
		"Books", "Music", "Movies", "Toys", "Games",
		"Clothing", "Shoes", "Jewelry", "Watches", "Bags",
		"Beauty", "Health", "Personal Care", "Baby", "Kids",
		"Automotive", "Tools", "Industrial", "Office", "School",
		"Pet Supplies", "Food", "Grocery", "Beverages", "Snacks",
		"Art", "Crafts", "Sewing", "Party", "Wedding",
		"Travel", "Luggage", "Outdoor Recreation", "Water Sports", "Snow Sports",
	}

	for i := 0; i < 100; i++ {
		categoryID := uint(i + 1)
		categoryName := categoryNames[i%len(categoryNames)]
		if i >= len(categoryNames) {
			categoryName = fmt.Sprintf("%s %d", categoryName, i/len(categoryNames)+1)
		}

		description := fmt.Sprintf("Category for %s products - Item %d", categoryName, i+1)
		categories = append(categories, entities.Category{
			ID:          categoryID,
			Name:        categoryName,
			Description: description,
		})
	}

	return categories
}

// generateExtensiveProducts creates a large number of products for performance testing
func generateExtensiveProducts(categories []entities.Category) []entities.Product {
	products := make([]entities.Product, 0, 10000)

	productPrefixes := []string{
		"Premium", "Deluxe", "Standard", "Basic", "Pro", "Elite", "Ultra",
		"Advanced", "Professional", "Classic", "Modern", "Vintage", "Limited Edition",
	}

	productTypes := []string{
		"Widget", "Gadget", "Device", "Tool", "Instrument", "Equipment", "Apparatus",
		"Machine", "Unit", "System", "Kit", "Set", "Package", "Bundle",
	}

	statuses := []entities.ProductStatus{
		entities.ProductStatusInStock,
		entities.ProductStatusInStock | entities.ProductStatusFeatured,
		entities.ProductStatusInStock | entities.ProductStatusOnSale,
		entities.ProductStatusDiscontinued,
	}

	cities := []string{
		"Seattle", "San Francisco", "New York", "Los Angeles", "Chicago",
		"Boston", "Austin", "Denver", "Portland", "Miami",
	}

	states := []string{
		"WA", "CA", "NY", "IL", "MA", "TX", "CO", "OR", "FL",
	}

	// Use a consistent seed for reproducible performance tests
	rng := rand.New(rand.NewSource(42))

	for i := 0; i < 10000; i++ {
		productID := uint(i + 1)
		categoryID := categories[rng.Intn(len(categories))].ID

		prefix := productPrefixes[i%len(productPrefixes)]
		productType := productTypes[(i/len(productPrefixes))%len(productTypes)]
		name := fmt.Sprintf("%s %s %d", prefix, productType, i+1)

		price := float64(rng.Intn(99900)+100) / 100.0 // Price between 1.00 and 1000.00
		status := statuses[rng.Intn(len(statuses))]

		city := cities[rng.Intn(len(cities))]
		state := states[rng.Intn(len(states))]

		createdAt := time.Now().AddDate(0, 0, -rng.Intn(365)) // Random date within last year

		products = append(products, entities.Product{
			ID:         productID,
			Name:       name,
			Price:      price,
			CategoryID: &categoryID,
			Status:     status,
			Version:    1,
			CreatedAt:  createdAt,
			ShippingAddress: &entities.Address{
				Street:     fmt.Sprintf("%d Main St", rng.Intn(9999)+1),
				City:       city,
				State:      state,
				PostalCode: fmt.Sprintf("%05d", rng.Intn(99999)),
				Country:    "USA",
			},
			Dimensions: &entities.Dimensions{
				Length: float64(rng.Intn(100) + 1),
				Width:  float64(rng.Intn(100) + 1),
				Height: float64(rng.Intn(100) + 1),
				Unit:   "cm",
			},
		})
	}

	return products
}

// generateExtensiveProductDescriptions creates product descriptions for performance testing
func generateExtensiveProductDescriptions(products []entities.Product) []entities.ProductDescription {
	descriptions := make([]entities.ProductDescription, 0, len(products)*3)

	languages := []string{"EN", "ES", "FR"}

	for _, product := range products {
		for _, lang := range languages {
			var description string
			switch lang {
			case "EN":
				description = fmt.Sprintf("This is a high-quality %s with excellent features and performance.", product.Name)
			case "ES":
				description = fmt.Sprintf("Este es un %s de alta calidad con excelentes características y rendimiento.", product.Name)
			case "FR":
				description = fmt.Sprintf("Ceci est un %s de haute qualité avec d'excellentes caractéristiques et performances.", product.Name)
			}

			descriptions = append(descriptions, entities.ProductDescription{
				ProductID:   product.ID,
				LanguageKey: lang,
				Description: description,
			})
		}
	}

	return descriptions
}

// registerReseedAction registers an unbound action to reseed the database
// This is useful for performance testing to reset the database to a known state
func registerReseedAction(service *odata.Service, db *gorm.DB) {
	if err := service.RegisterAction(odata.ActionDefinition{
		Name:       "Reseed",
		IsBound:    false,
		Parameters: []odata.ParameterDefinition{},
		ReturnType: nil,
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) error {
			// Reseed the database with extensive data
			if err := seedDatabase(db, true); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return err
			}

			// Return success response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			response := map[string]interface{}{
				"status":  "success",
				"message": "Database reseeded with extensive performance testing data",
			}

			return json.NewEncoder(w).Encode(response)
		},
	}); err != nil {
		panic("Failed to register Reseed action: " + err.Error())
	}
}
