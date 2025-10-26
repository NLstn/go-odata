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
	"time"

	"github.com/NLstn/go-odata/perfserver/entities"
	"github.com/nlstn/go-odata"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Db is the global database instance used throughout the performance server
var Db *gorm.DB

func main() {
	// Parse command-line flags
	dbType := flag.String("db", "sqlite", "Database type: sqlite or postgres")
	dbDSN := flag.String("dsn", "", "Database DSN (connection string). For postgres, use postgresql://... format. For sqlite, use file path or :memory:")
	port := flag.String("port", "9091", "Port to listen on")
	cpuProfile := flag.String("cpuprofile", "", "Write CPU profile to file")
	traceSQL := flag.Bool("trace-sql", false, "Enable SQL query tracing and optimization analysis")
	traceSQLFile := flag.String("trace-sql-file", "", "Write SQL trace analysis to file (requires --trace-sql)")
	extensive := flag.Bool("extensive", true, "Use extensive seeding with large datasets for performance testing")
	flag.Parse()

	// Create SQL tracer if requested (declared early so signal handler can access it)
	var sqlTracer *SQLTracer
	if *traceSQL {
		sqlTracer = NewSQLTracer(100*time.Millisecond, false)
		WriteQueryLogHeader()
	}

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
	}

	// Set up signal handler for both CPU profiling and SQL tracing
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n⏸️  Received interrupt signal, shutting down...")

		// Stop CPU profiling if enabled
		if *cpuProfile != "" {
			pprof.StopCPUProfile()
			fmt.Println("CPU profile written to:", *cpuProfile)
		}

		// Print SQL trace summary if enabled
		if sqlTracer != nil {
			sqlTracer.PrintSummary()
			if *traceSQLFile != "" {
				if err := sqlTracer.ExportToFile(*traceSQLFile); err != nil {
					log.Printf("Error writing SQL trace file: %v", err)
				} else {
					fmt.Printf("📝 SQL trace analysis written to: %s\n", *traceSQLFile)
				}
			}
		}

		os.Exit(0)
	}()

	// Determine database configuration
	var err error

	switch *dbType {
	case "sqlite":
		dsn := ":memory:"
		if *dbDSN != "" {
			dsn = *dbDSN
		}

		config := &gorm.Config{}
		if sqlTracer != nil {
			config.Logger = sqlTracer
		}

		Db, err = gorm.Open(sqlite.Open(dsn), config)
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

		config := &gorm.Config{}
		if sqlTracer != nil {
			config.Logger = sqlTracer
		}

		Db, err = gorm.Open(postgres.Open(dsn), config)
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
	fmt.Println("🌱 Seeding database...")
	startSeed := time.Now()
	if err := seedDatabase(Db, *extensive); err != nil {
		log.Fatal("Failed to seed database:", err)
	}
	seedDuration := time.Since(startSeed)
	fmt.Printf("✅ Database seeding completed in %.2f seconds\n", seedDuration.Seconds())

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

	// Register functions for performance testing
	registerFunctions(service, Db)

	// Register actions for performance testing
	registerActions(service, Db)

	// Register reseed action for performance testing
	registerReseedAction(service, Db)

	// Create HTTP mux and register the OData service
	mux := http.NewServeMux()
	mux.Handle("/", service)

	// Start the HTTP server (no middleware for performance server to reduce overhead)
	fmt.Println("🚀 Performance testing server starting...")
	fmt.Println("Service endpoints:")
	fmt.Printf("  Service Document:     http://localhost:%s/\n", *port)
	fmt.Printf("  Metadata (XML):       http://localhost:%s/$metadata\n", *port)
	fmt.Printf("  Metadata (JSON):      http://localhost:%s/$metadata?$format=json\n", *port)
	fmt.Printf("  Categories:           http://localhost:%s/Categories\n", *port)
	fmt.Printf("  Products:             http://localhost:%s/Products\n", *port)
	fmt.Printf("  ProductDescriptions:  http://localhost:%s/ProductDescriptions\n", *port)
	fmt.Printf("  Company (Singleton):  http://localhost:%s/Company\n", *port)
	fmt.Println()
	fmt.Println("Performance Testing:")
	fmt.Printf("  POST http://localhost:%s/Reseed  (Resets database to performance testing state)\n", *port)
	fmt.Println()

	if *traceSQL {
		fmt.Println("🔍 SQL Query Tracing: ENABLED")
		fmt.Println("  Slow query threshold: 100ms")
		if *traceSQLFile != "" {
			fmt.Printf("  Trace output file: %s\n", *traceSQLFile)
		}
		fmt.Println()
	}

	if *cpuProfile != "" {
		fmt.Println("📊 CPU Profiling: ENABLED")
		fmt.Printf("  Profile output file: %s\n", *cpuProfile)
		fmt.Println()
	}

	if *extensive {
		fmt.Println("📈 Extensive seeding mode: ENABLED")
		fmt.Println("  10,000 products, 100 categories, 30,000 descriptions")
		fmt.Println()
	}

	if err := http.ListenAndServe(":"+*port, mux); err != nil {
		log.Fatal("Server failed:", err)
	}

	// Print SQL summary when server exits normally
	if sqlTracer != nil {
		sqlTracer.PrintSummary()
		if *traceSQLFile != "" {
			if err := sqlTracer.ExportToFile(*traceSQLFile); err != nil {
				log.Printf("Error writing SQL trace file: %v", err)
			} else {
				fmt.Printf("📝 SQL trace analysis written to: %s\n", *traceSQLFile)
			}
		}
	}
}
