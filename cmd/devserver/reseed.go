package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/NLstn/go-odata/devserver/entities"
	"github.com/nlstn/go-odata"
	"gorm.io/gorm"
)

// seedDatabase initializes the database with sample data
// This function drops and recreates all tables to ensure a clean state
func seedDatabase(db *gorm.DB) error {
	// Get the database dialect name to determine the database type
	dialectName := db.Name()

	// Drop all tables (GORM handles the correct order based on foreign keys)
	if err := db.Migrator().DropTable(&entities.ProductDescription{}, &entities.Product{}, &entities.Category{}, &entities.CompanyInfo{}, &entities.User{}, &entities.APIKey{}); err != nil {
		return fmt.Errorf("failed to drop tables: %w", err)
	}

	// For PostgreSQL, explicitly reset sequences after dropping tables
	// This ensures auto-increment columns start from 1 after reseeding
	// Only drop sequences for tables that have auto-increment integer primary keys
	if dialectName == "postgres" {
		// List of table names that need sequence resets (tables with auto-increment primary keys)
		// Excludes tables with composite keys (product_descriptions) or UUID keys (api_keys)
		tables := []string{"categories", "products", "company_infos", "users"}
		for _, table := range tables {
			// PostgreSQL convention: sequence name is table_column_seq
			// GORM uses lowercase table names and "id" for primary key columns by default
			var sequenceName string
			if table == "users" {
				sequenceName = table + "_user_id_seq" // Users table has user_id column
			} else {
				sequenceName = table + "_id_seq"
			}
			if err := db.Exec(fmt.Sprintf("DROP SEQUENCE IF EXISTS %s CASCADE", sequenceName)).Error; err != nil {
				// Log but don't fail - sequence might not exist
				fmt.Printf("Note: Could not drop sequence %s: %v\n", sequenceName, err)
			}
		}
	}

	// Recreate tables with fresh schema (auto-increment counters are automatically reset)
	if err := db.AutoMigrate(&entities.Category{}, &entities.Product{}, &entities.ProductDescription{}, &entities.CompanyInfo{}, &entities.User{}, &entities.APIKey{}); err != nil {
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

	// Seed users
	sampleUsers := entities.GetSampleUsers()
	if err := db.Create(&sampleUsers).Error; err != nil {
		return fmt.Errorf("failed to seed users: %w", err)
	}

	sampleAPIKeys := entities.GetSampleAPIKeys()
	if err := db.Create(&sampleAPIKeys).Error; err != nil {
		return fmt.Errorf("failed to seed API keys: %w", err)
	}

	// For PostgreSQL, reset sequence values to match the max ID in each table
	// This ensures that the next auto-generated ID doesn't conflict with existing IDs
	// Only reset sequences for tables that have auto-increment integer primary keys
	if dialectName == "postgres" {
		// Map of table names to their sequence names and column names
		// Only includes tables with auto-increment IDs
		sequenceResets := map[string]struct {
			sequence string
			column   string
		}{
			"categories":    {sequence: "categories_id_seq", column: "id"},
			"products":      {sequence: "products_id_seq", column: "id"},
			"company_infos": {sequence: "company_infos_id_seq", column: "id"},
			"users":         {sequence: "users_user_id_seq", column: "user_id"},
		}

		for table, info := range sequenceResets {
			// Set sequence value to MAX(column) + 1 from the table
			resetSQL := fmt.Sprintf(
				"SELECT setval('%s', COALESCE((SELECT MAX(%s) FROM %s), 0) + 1, false)",
				info.sequence, info.column, table,
			)
			if err := db.Exec(resetSQL).Error; err != nil {
				// Log but don't fail - just a warning
				fmt.Printf("Warning: Could not reset sequence %s: %v\n", info.sequence, err)
			}
		}
	}

	fmt.Printf("Database seeded with %d categories, %d products, %d descriptions, %d users, %d API keys, and company info\n",
		len(sampleCategories), len(sampleProducts), len(sampleDescriptions), len(sampleUsers), len(sampleAPIKeys))
	return nil
}

// registerReseedAction registers an unbound action to reseed the database
// This is useful for testing to reset the database to a known state
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
