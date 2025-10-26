package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"

	"github.com/NLstn/go-odata/complianceserver/entities"
	"github.com/nlstn/go-odata"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Db is the global database instance used throughout the compliance server
var Db *gorm.DB

func main() {
	// Parse command-line flags
	dbType := flag.String("db", "sqlite", "Database type: sqlite or postgres")
	dbDSN := flag.String("dsn", "", "Database DSN (connection string). For postgres, use postgresql://... format. For sqlite, use file path or :memory:")
	port := flag.String("port", "9090", "Port to listen on")
	cpuProfile := flag.String("cpuprofile", "", "Write CPU profile to file")
	flag.Parse()

	// Start CPU profiling if requested
	if *cpuProfile != "" {
		f, err := os.Create(*cpuProfile)
		if err != nil {
			log.Fatal("Could not create CPU profile: ", err)
		}
		defer func() {
			if err := f.Close(); err != nil {
				log.Printf("Error closing CPU profile file: %v", err)
			}
		}()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("Could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
		fmt.Printf("📊 CPU profiling enabled, writing to: %s\n", *cpuProfile)

		// Set up signal handler to ensure profile is written on interrupt
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-sigChan
			fmt.Println("\n⏸️  Received interrupt signal, stopping CPU profiler...")
			pprof.StopCPUProfile()
			if err := f.Close(); err != nil {
				log.Printf("Error closing CPU profile file: %v", err)
			}
			fmt.Println("CPU profile written to:", *cpuProfile)
			os.Exit(0)
		}()
	}

	// Determine database configuration
	var err error

	switch *dbType {
	case "sqlite":
		dsn := ":memory:"
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

	// Auto-migrate the Product, ProductDescription, Category, and CompanyInfo models
	if err := Db.AutoMigrate(&entities.Category{}, &entities.Product{}, &entities.ProductDescription{}, &entities.CompanyInfo{}); err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	// Seed the database with sample data
	if err := seedDatabase(Db); err != nil {
		log.Fatal("Failed to seed database:", err)
	}

	// Create OData service
	service := odata.NewService(Db)

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

	// Register the CompanyInfo singleton
	if err := service.RegisterSingleton(&entities.CompanyInfo{}, "Company"); err != nil {
		log.Fatal("Failed to register Company singleton:", err)
	}

	// Register functions for compliance testing
	registerFunctions(service, Db)

	// Register actions for compliance testing
	registerActions(service, Db)

	// Register reseed action for compliance testing
	registerReseedAction(service, Db)

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
	fmt.Println()
	fmt.Println("Compliance Testing:")
	fmt.Printf("  POST http://localhost:%s/Reseed  (Resets database to default state)\n", *port)
	fmt.Println()

	if err := http.ListenAndServe(":"+*port, mux); err != nil {
		log.Fatal("Server failed:", err)
	}
}
