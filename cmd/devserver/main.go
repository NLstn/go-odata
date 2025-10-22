package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/nlstn/go-odata"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// seedDatabase initializes the database with sample data
// This function clears all existing data and resets to the default state
func seedDatabase(db *gorm.DB, dbType string) error {
	// Clear existing data
	if err := db.Exec("DELETE FROM product_descriptions").Error; err != nil {
		return fmt.Errorf("failed to clear product descriptions: %w", err)
	}
	if err := db.Exec("DELETE FROM product_relations").Error; err != nil {
		// This is a many-to-many table, so it might not exist on first run
		log.Printf("Info: Could not clear product relations: %v", err)
	}
	if err := db.Exec("DELETE FROM products").Error; err != nil {
		return fmt.Errorf("failed to clear products: %w", err)
	}
	if err := db.Exec("DELETE FROM categories").Error; err != nil {
		return fmt.Errorf("failed to clear categories: %w", err)
	}
	if err := db.Exec("DELETE FROM company_infos").Error; err != nil {
		return fmt.Errorf("failed to clear company info: %w", err)
	}

	// Reset auto-increment counters (database specific)
	switch dbType {
	case "sqlite":
		if err := db.Exec("DELETE FROM sqlite_sequence WHERE name IN ('products', 'product_descriptions', 'company_infos')").Error; err != nil {
			// This is not critical, just continue
			log.Printf("Warning: Could not reset auto-increment counters: %v", err)
		}
	case "postgres":
		// PostgreSQL: reset sequences
		sequences := []string{"products_id_seq", "company_infos_id_seq"}
		for _, seq := range sequences {
			if err := db.Exec(fmt.Sprintf("ALTER SEQUENCE IF EXISTS %s RESTART WITH 1", seq)).Error; err != nil {
				log.Printf("Warning: Could not reset sequence %s: %v", seq, err)
			}
		}
	}

	// Seed categories first (products reference categories)
	sampleCategories := GetSampleCategories()
	if err := db.Create(&sampleCategories).Error; err != nil {
		return fmt.Errorf("failed to seed categories: %w", err)
	}

	// Seed products
	sampleProducts := GetSampleProducts()
	if err := db.Create(&sampleProducts).Error; err != nil {
		return fmt.Errorf("failed to seed products: %w", err)
	}

	// Seed product descriptions
	sampleDescriptions := GetSampleProductDescriptions()
	if err := db.Create(&sampleDescriptions).Error; err != nil {
		return fmt.Errorf("failed to seed product descriptions: %w", err)
	}

	// Seed company info singleton
	companyInfo := GetCompanyInfo()
	if err := db.Create(&companyInfo).Error; err != nil {
		return fmt.Errorf("failed to seed company info: %w", err)
	}

	fmt.Printf("Database seeded with %d categories, %d products, %d descriptions, and company info\n",
		len(sampleCategories), len(sampleProducts), len(sampleDescriptions))
	return nil
}

func main() {
	// Parse command-line flags
	dbType := flag.String("db", "sqlite", "Database type: sqlite or postgres")
	dbDSN := flag.String("dsn", "", "Database DSN (connection string). For postgres, use postgresql://... format. For sqlite, use file path or :memory:")
	flag.Parse()

	// Determine database configuration
	var db *gorm.DB
	var err error

	switch *dbType {
	case "sqlite":
		dsn := ":memory:"
		if *dbDSN != "" {
			dsn = *dbDSN
		}
		db, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{})
		if err != nil {
			log.Fatal("Failed to connect to SQLite database:", err)
		}
		fmt.Println("📦 Using SQLite database:", dsn)

	case "postgres":
		dsn := *dbDSN
		if dsn == "" {
			// Check for environment variable as fallback
			dsn = os.Getenv("DATABASE_URL")
			if dsn == "" {
				log.Fatal("PostgreSQL DSN required. Use -dsn flag or set DATABASE_URL environment variable")
			}
		}
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			log.Fatal("Failed to connect to PostgreSQL database:", err)
		}
		fmt.Println("🐘 Using PostgreSQL database")

	default:
		log.Fatalf("Unsupported database type: %s. Use 'sqlite' or 'postgres'", *dbType)
	}

	// Auto-migrate the Product, ProductDescription, Category, and CompanyInfo models
	if err := db.AutoMigrate(&Category{}, &Product{}, &ProductDescription{}, &CompanyInfo{}); err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	// Seed the database with sample data
	if err := seedDatabase(db, *dbType); err != nil {
		log.Fatal("Failed to seed database:", err)
	}

	// Create OData service
	service := odata.NewService(db)

	// Register the Category, Product and ProductDescription entities
	if err := service.RegisterEntity(&Category{}); err != nil {
		log.Fatal("Failed to register Category entity:", err)
	}
	if err := service.RegisterEntity(&Product{}); err != nil {
		log.Fatal("Failed to register Product entity:", err)
	}
	if err := service.RegisterEntity(&ProductDescription{}); err != nil {
		log.Fatal("Failed to register ProductDescription entity:", err)
	}

	// Register the CompanyInfo singleton
	if err := service.RegisterSingleton(&CompanyInfo{}, "Company"); err != nil {
		log.Fatal("Failed to register Company singleton:", err)
	}

	// Register example functions
	registerFunctions(service, db)

	// Register example actions
	registerActions(service, db)

	// Register reseed action for testing
	registerReseedAction(service, db, *dbType)

	// Start the HTTP server
	fmt.Println("🚀 Development server starting with hot reload...")
	fmt.Println("Service endpoints:")
	fmt.Println("  Service Document:     http://localhost:8080/")
	fmt.Println("  Metadata (XML):       http://localhost:8080/$metadata")
	fmt.Println("  Metadata (JSON):      http://localhost:8080/$metadata?$format=json")
	fmt.Println("  Categories:           http://localhost:8080/Categories")
	fmt.Println("  Single Category:      http://localhost:8080/Categories(1)")
	fmt.Println("  Products:             http://localhost:8080/Products")
	fmt.Println("  Single Product:       http://localhost:8080/Products(1)")
	fmt.Println("  ProductDescriptions:  http://localhost:8080/ProductDescriptions")
	fmt.Println("  Product Descriptions: http://localhost:8080/ProductDescriptions(ProductID=1,LanguageKey='EN')")
	fmt.Println("  Company (Singleton):  http://localhost:8080/Company")
	fmt.Println()
	fmt.Println("OData Actions and Functions:")
	fmt.Println("  Unbound Functions:")
	fmt.Println("    GET  http://localhost:8080/GetTopProducts?count=3")
	fmt.Println("    GET  http://localhost:8080/GetProductStats")
	fmt.Println("  Bound Functions:")
	fmt.Println("    GET  http://localhost:8080/Products(1)/GetTotalPrice?taxRate=0.08")
	fmt.Println("  Unbound Actions:")
	fmt.Println("    POST http://localhost:8080/ResetAllPrices")
	fmt.Println("    POST http://localhost:8080/Reseed  (Test utility - resets database to default state)")
	fmt.Println("  Bound Actions:")
	fmt.Println("    POST http://localhost:8080/Products(1)/ApplyDiscount (body: {\"percentage\": 10})")
	fmt.Println("    POST http://localhost:8080/Products(1)/IncreasePrice (body: {\"amount\": 5.0})")
	fmt.Println()

	if err := service.ListenAndServe(":8080"); err != nil {
		log.Fatal("Server failed:", err)
	}
}
