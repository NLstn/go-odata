package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/nlstn/go-odata"
	"github.com/nlstn/go-odata/cmd/complianceserver/entities"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Db is the global database instance used throughout the compliance server
var Db *gorm.DB

func main() {
	// Parse command-line flags
	dbType := flag.String("db", "sqlite", "Database type: sqlite or postgres")
	dbDSN := flag.String("dsn", "", "Database DSN (connection string). For postgres, use postgresql://... format. For sqlite, use file path")
	port := flag.String("port", "9090", "Port to listen on")
	flag.Parse()

	// Determine database configuration
	var err error

	switch *dbType {
	case "sqlite":
		dsn := "/tmp/go-odata-compliance.db"
		if *dbDSN != "" {
			dsn = *dbDSN
		}

		Db, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{})
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

		Db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			log.Fatal("Failed to connect to PostgreSQL database:", err)
		}
		fmt.Println("🐘 Using PostgreSQL database")

	default:
		log.Fatalf("Unsupported database type: %s. Use 'sqlite' or 'postgres'", *dbType)
	}

	// Initialize database with sample data
	// This handles the initial migration and seeding in one step
	if err := seedDatabase(Db); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	// Create OData service
	service := odata.NewService(Db)

	if err := service.SetNamespace("ComplianceService"); err != nil {
		log.Fatal("Failed to set service namespace:", err)
	}

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

	if err := service.EnableChangeTracking("Products"); err != nil {
		log.Fatal("Failed to enable change tracking for Products:", err)
	}

	// Register the CompanyInfo singleton
	if err := service.RegisterSingleton(&entities.CompanyInfo{}, "Company"); err != nil {
		log.Fatal("Failed to register Company singleton:", err)
	}

	// Register the MediaItem entity for media entity compliance testing
	if err := service.RegisterEntity(&entities.MediaItem{}); err != nil {
		log.Fatal("Failed to register MediaItem entity:", err)
	}

	// Register functions for compliance testing
	registerFunctions(service, Db)

	// Register actions for compliance testing
	registerActions(service, Db)

	// Register reseed action for compliance testing
	registerReseedAction(service, Db)

	// Enable asynchronous processing for compliance testing
	if err := service.EnableAsyncProcessing(odata.AsyncConfig{
		MonitorPathPrefix:    "/$async/jobs/",
		DefaultRetryInterval: 2 * time.Second,
		JobRetention:         5 * time.Minute,
	}); err != nil {
		log.Fatal("Failed to enable async processing:", err)
	}

	// Create HTTP mux and register the OData service
	mux := http.NewServeMux()
	mux.Handle("/", service)

	// Start the HTTP server (no middleware for compliance server)
	fmt.Println("🚀 Compliance server starting...")
	fmt.Println("Service endpoints:")
	fmt.Printf("  Service Document:     http://localhost:%s/\n", *port)
	fmt.Printf("  Metadata (XML):       http://localhost:%s/$metadata\n", *port)
	fmt.Printf("  Metadata (JSON):      http://localhost:%s/$metadata?$format=json\n", *port)
	fmt.Printf("  Categories:           http://localhost:%s/Categories\n", *port)
	fmt.Printf("  Products:             http://localhost:%s/Products\n", *port)
	fmt.Printf("  ProductDescriptions:  http://localhost:%s/ProductDescriptions\n", *port)
	fmt.Printf("  Company (Singleton):  http://localhost:%s/Company\n", *port)
	fmt.Printf("  Async Monitor:        http://localhost:%s/$async/jobs/{jobID}\n", *port)
	fmt.Println()
	fmt.Println("Compliance Testing:")
	fmt.Printf("  POST http://localhost:%s/Reseed  (Resets database to default state)\n", *port)
	fmt.Println()
	fmt.Println("Asynchronous Processing:")
	fmt.Println("  Use 'Prefer: respond-async' header to enable async processing")
	fmt.Println("  Status monitors available at /$async/jobs/{jobID}")
	fmt.Println()

	if err := http.ListenAndServe(":"+*port, mux); err != nil {
		log.Fatal("Server failed:", err)
	}
}
