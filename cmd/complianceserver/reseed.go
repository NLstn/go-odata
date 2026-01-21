package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/nlstn/go-odata"
	"github.com/nlstn/go-odata/cmd/complianceserver/entities"
	"gorm.io/gorm"
)

// seedDatabase initializes the database with sample data
// This function drops and recreates all tables to ensure a clean state
// It handles both initial setup and reseeding operations
func seedDatabase(db *gorm.DB) error {
	// Get the database dialect name to determine the database type
	dialectName := db.Name()

	// Drop all existing tables to ensure a clean state
	// For PostgreSQL, we need to handle join tables explicitly first
	if dialectName == "postgres" {
		// Drop the many-to-many join table first to avoid foreign key constraint issues
		// Intentionally ignoring errors if the table doesn't exist (first-time setup)
		//nolint:errcheck
		_ = db.Migrator().DropTable("product_relations")
	}

	// Drop all entity tables in reverse dependency order
	// ProductDescription -> Product -> Category (and CompanyInfo, MediaItem independently)
	// Intentionally ignoring errors if tables don't exist (first-time setup)
	//nolint:errcheck
	_ = db.Migrator().DropTable(&entities.ProductDescription{})
	//nolint:errcheck
	_ = db.Migrator().DropTable(&entities.ReadOnlyItem{})
	//nolint:errcheck
	_ = db.Migrator().DropTable(&entities.Product{})
	//nolint:errcheck
	_ = db.Migrator().DropTable(&entities.Category{})
	//nolint:errcheck
	_ = db.Migrator().DropTable(&entities.CompanyInfo{})
	//nolint:errcheck
	_ = db.Migrator().DropTable(&entities.MediaItem{})

	// Drop FTS tables if they exist to ensure search indexes are rebuilt
	// Only for SQLite - FTS tables are virtual tables specific to SQLite
	if dialectName == "sqlite" {
		//nolint:errcheck
		_ = db.Exec("DROP TABLE IF EXISTS Products_fts").Error
		//nolint:errcheck
		_ = db.Exec("DROP TABLE IF EXISTS Categories_fts").Error
		//nolint:errcheck
		_ = db.Exec("DROP TABLE IF EXISTS ProductDescriptions_fts").Error
		//nolint:errcheck
		_ = db.Exec("DROP TABLE IF EXISTS CompanyInfo_fts").Error
		//nolint:errcheck
		_ = db.Exec("DROP TABLE IF EXISTS MediaItems_fts").Error
	}

	// Recreate tables with fresh schema (auto-increment counters are automatically reset)
	if err := db.AutoMigrate(&entities.Category{}, &entities.Product{}, &entities.ProductDescription{}, &entities.CompanyInfo{}, &entities.MediaItem{}, &entities.ReadOnlyItem{}); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	// Seed categories first (products reference categories)
	sampleCategories := entities.GetSampleCategories()
	// Generate UUIDs for categories
	for i := range sampleCategories {
		sampleCategories[i].ID = uuid.New()
	}
	if err := db.Create(&sampleCategories).Error; err != nil {
		return fmt.Errorf("failed to seed categories: %w", err)
	}

	// Seed products and set up relationships with categories
	sampleProducts := entities.GetSampleProducts()
	// Generate UUIDs for products and assign category IDs
	// Products 0, 1, 4, 5, 6 -> Electronics (index 0)
	// Product 2 -> Kitchen (index 1)
	// Product 3 -> Furniture (index 2)
	for i := range sampleProducts {
		sampleProducts[i].ID = uuid.New()
	}
	if len(sampleCategories) >= 3 && len(sampleProducts) >= 7 {
		sampleProducts[0].CategoryID = &sampleCategories[0].ID // Laptop -> Electronics
		sampleProducts[1].CategoryID = &sampleCategories[0].ID // Wireless Mouse -> Electronics
		sampleProducts[2].CategoryID = &sampleCategories[1].ID // Coffee Mug -> Kitchen
		sampleProducts[3].CategoryID = &sampleCategories[2].ID // Office Chair -> Furniture
		sampleProducts[4].CategoryID = &sampleCategories[0].ID // Smartphone -> Electronics
		sampleProducts[5].CategoryID = &sampleCategories[0].ID // Premium Laptop Pro -> Electronics
		sampleProducts[6].CategoryID = &sampleCategories[0].ID // Gaming Mouse Ultra -> Electronics
	}

	if err := db.Create(&sampleProducts).Error; err != nil {
		return fmt.Errorf("failed to seed products: %w", err)
	}

	// Seed product descriptions and link them to products
	sampleDescriptions := entities.GetSampleProductDescriptions()
	// Map descriptions to products based on their original order
	// Descriptions 0, 1 -> Product 0 (Laptop)
	// Descriptions 2, 3 -> Product 1 (Wireless Mouse)
	// Description 4 -> Product 2 (Coffee Mug)
	// Descriptions 5, 6 -> Product 4 (Smartphone)
	if len(sampleProducts) >= 5 && len(sampleDescriptions) >= 7 {
		sampleDescriptions[0].ProductID = sampleProducts[0].ID // EN for Laptop
		sampleDescriptions[1].ProductID = sampleProducts[0].ID // DE for Laptop
		sampleDescriptions[2].ProductID = sampleProducts[1].ID // EN for Wireless Mouse
		sampleDescriptions[3].ProductID = sampleProducts[1].ID // FR for Wireless Mouse
		sampleDescriptions[4].ProductID = sampleProducts[2].ID // EN for Coffee Mug
		sampleDescriptions[5].ProductID = sampleProducts[4].ID // EN for Smartphone
		sampleDescriptions[6].ProductID = sampleProducts[4].ID // ES for Smartphone
	}

	if err := db.Create(&sampleDescriptions).Error; err != nil {
		return fmt.Errorf("failed to seed product descriptions: %w", err)
	}

	// Seed company info singleton
	companyInfo := entities.GetCompanyInfo()
	companyInfo.ID = uuid.New()
	if err := db.Create(&companyInfo).Error; err != nil {
		return fmt.Errorf("failed to seed company info: %w", err)
	}

	// Seed media items
	sampleMediaItems := getSampleMediaItems()
	// Generate UUIDs for media items
	for i := range sampleMediaItems {
		sampleMediaItems[i].ID = uuid.New()
	}
	if err := db.Create(&sampleMediaItems).Error; err != nil {
		return fmt.Errorf("failed to seed media items: %w", err)
	}

	// Seed read-only items
	sampleReadOnlyItems := entities.GetSampleReadOnlyItems()
	for i := range sampleReadOnlyItems {
		sampleReadOnlyItems[i].ID = uuid.New()
	}
	if err := db.Create(&sampleReadOnlyItems).Error; err != nil {
		return fmt.Errorf("failed to seed read-only items: %w", err)
	}

	fmt.Printf("Database seeded with %d categories, %d products, %d descriptions, %d media items, %d read-only items, and company info\n",
		len(sampleCategories), len(sampleProducts), len(sampleDescriptions), len(sampleMediaItems), len(sampleReadOnlyItems))
	return nil
}

// getSampleMediaItems returns sample media items for seeding
// Note: IDs are server-generated
func getSampleMediaItems() []entities.MediaItem {
	now := time.Now()
	size1 := int64(1024)
	size2 := int64(2048)
	return []entities.MediaItem{
		{
			Name:        "Sample Image",
			ContentType: "image/png",
			Size:        &size1,
			Content:     []byte("fake-png-binary-data"),
			CreatedAt:   now,
			ModifiedAt:  now,
		},
		{
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

			// Clear FTS cache after dropping FTS tables during seedDatabase
			// This ensures FTS tables will be recreated when needed
			service.ResetFTS()

			// Drop the async jobs table if it exists to ensure clean state
			if err := db.Migrator().DropTable("_odata_async_jobs"); err != nil {
				// Log but don't fail - table might not exist
				fmt.Printf("Note: Could not drop async jobs table: %v\n", err)
			}

			// Re-enable async processing to recreate the _odata_async_jobs table
			// that was dropped during seedDatabase
			if err := service.EnableAsyncProcessing(odata.AsyncConfig{
				MonitorPathPrefix:    "/$async/jobs/",
				DefaultRetryInterval: 2 * time.Second,
				JobRetention:         5 * time.Minute,
			}); err != nil {
				http.Error(w, fmt.Sprintf("failed to re-enable async processing: %v", err), http.StatusInternalServerError)
				return err
			}

			// Wait a brief moment to ensure async table is fully initialized
			// This prevents race conditions when tests immediately use async features
			time.Sleep(50 * time.Millisecond)

			// Verify async table was created successfully
			if db.Migrator().HasTable("_odata_async_jobs") {
				fmt.Println("Async jobs table successfully initialized")
			} else {
				fmt.Println("Warning: Async jobs table not detected after initialization")
			} // Return success response
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
