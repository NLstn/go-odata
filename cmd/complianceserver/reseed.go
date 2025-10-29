package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/nlstn/go-odata"
	"github.com/nlstn/go-odata/cmd/complianceserver/entities"
	"gorm.io/gorm"
)

// seedDatabase initializes the database with sample data
// This function drops and recreates all tables to ensure a clean state
func seedDatabase(db *gorm.DB) error {
	// Drop all tables (GORM handles the correct order based on foreign keys)
	if err := db.Migrator().DropTable(&entities.ProductDescription{}, &entities.Product{}, &entities.Category{}, &entities.CompanyInfo{}, &entities.MediaItem{}); err != nil {
		return fmt.Errorf("failed to drop tables: %w", err)
	}

	// Recreate tables with fresh schema (auto-increment counters are automatically reset)
	if err := db.AutoMigrate(&entities.Category{}, &entities.Product{}, &entities.ProductDescription{}, &entities.CompanyInfo{}, &entities.MediaItem{}); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	// Seed categories first (products reference categories)
	sampleCategories := entities.GetSampleCategories()
	if err := db.Create(&sampleCategories).Error; err != nil {
		return fmt.Errorf("failed to seed categories: %w", err)
	}

	// Seed products
	sampleProducts := entities.GetSampleProducts()
	if err := db.Create(&sampleProducts).Error; err != nil {
		return fmt.Errorf("failed to seed products: %w", err)
	}

	// Seed product descriptions
	sampleDescriptions := entities.GetSampleProductDescriptions()
	if err := db.Create(&sampleDescriptions).Error; err != nil {
		return fmt.Errorf("failed to seed product descriptions: %w", err)
	}

	// Seed company info singleton
	companyInfo := entities.GetCompanyInfo()
	if err := db.Create(&companyInfo).Error; err != nil {
		return fmt.Errorf("failed to seed company info: %w", err)
	}

	// Seed media items
	sampleMediaItems := getSampleMediaItems()
	if err := db.Create(&sampleMediaItems).Error; err != nil {
		return fmt.Errorf("failed to seed media items: %w", err)
	}

	fmt.Printf("Database seeded with %d categories, %d products, %d descriptions, %d media items, and company info\n",
		len(sampleCategories), len(sampleProducts), len(sampleDescriptions), len(sampleMediaItems))
	return nil
}

// getSampleMediaItems returns sample media items for seeding
func getSampleMediaItems() []entities.MediaItem {
	now := time.Now()
	size1 := int64(1024)
	size2 := int64(2048)
	return []entities.MediaItem{
		{
			ID:          1,
			Name:        "Sample Image",
			ContentType: "image/png",
			Size:        &size1,
			Content:     []byte("fake-png-binary-data"),
			CreatedAt:   now,
			ModifiedAt:  now,
		},
		{
			ID:          2,
			Name:        "Sample Document",
			ContentType: "application/pdf",
			Size:        &size2,
			Content:     []byte("fake-pdf-binary-data"),
			CreatedAt:   now,
			ModifiedAt:  now,
		},
	}
}

// registerReseedAction registers an unbound action to reseed the database
// This is useful for compliance testing to reset the database to a known state
func registerReseedAction(service *odata.Service, db *gorm.DB) {
	if err := service.RegisterAction(odata.ActionDefinition{
		Name:       "Reseed",
		IsBound:    false,
		Parameters: []odata.ParameterDefinition{},
		ReturnType: nil,
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) error {
			// Reseed the database
			if err := seedDatabase(db); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return err
			}

			// Return success response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			response := map[string]interface{}{
				"status":  "success",
				"message": "Database reseeded with default data",
			}

			return json.NewEncoder(w).Encode(response)
		},
	}); err != nil {
		panic("Failed to register Reseed action: " + err.Error())
	}
}
