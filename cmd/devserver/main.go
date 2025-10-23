package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/NLstn/go-odata/devserver/entities"
	"github.com/nlstn/go-odata"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

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

	// Auto-migrate the Product, ProductDescription, Category, CompanyInfo, and User models
	if err := db.AutoMigrate(&entities.Category{}, &entities.Product{}, &entities.ProductDescription{}, &entities.CompanyInfo{}, &entities.User{}); err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	// Seed the database with sample data
	if err := seedDatabase(db); err != nil {
		log.Fatal("Failed to seed database:", err)
	}

	// Create OData service
	service := odata.NewService(db)

	// Register the Category, Product and ProductDescription entities
	if err := service.RegisterEntity(&entities.Category{}); err != nil {
		log.Fatal("Failed to register Category entity:", err)
	}
	if err := service.RegisterEntity(&entities.Product{}); err != nil {
		log.Fatal("Failed to register Product entity:", err)
	}
	if err := service.RegisterEntity(&entities.ProductDescription{}); err != nil {
		log.Fatal("Failed to register ProductDescription entity:", err)
	}
	if err := service.RegisterEntity(&entities.User{}); err != nil {
		log.Fatal("Failed to register User entity:", err)
	}

	// Register the CompanyInfo singleton
	if err := service.RegisterSingleton(&entities.CompanyInfo{}, "Company"); err != nil {
		log.Fatal("Failed to register Company singleton:", err)
	}

	// Register example functions
	registerFunctions(service, db)

	// Register example actions
	registerActions(service, db)

	// Register reseed action for testing
	registerReseedAction(service, db)

	// Create HTTP mux and register the OData service
	mux := http.NewServeMux()
	mux.Handle("/", service)

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
	fmt.Println("  Users:                http://localhost:8080/Users")
	fmt.Println("  Single User:          http://localhost:8080/Users(1)")
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

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal("Server failed:", err)
	}
}
