package main

import (
	"fmt"
	"log"

	"github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	// Initialize SQLite in-memory database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto-migrate the Product model
	if err := db.AutoMigrate(&Product{}); err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	// Seed the database with sample data
	sampleProducts := GetSampleProducts()
	if err := db.Create(&sampleProducts).Error; err != nil {
		log.Fatal("Failed to seed database:", err)
	}

	fmt.Printf("Database initialized with %d products\n", len(sampleProducts))

	// Create OData service
	service := odata.NewService(db)

	// Register the Product entity
	if err := service.RegisterEntity(&Product{}); err != nil {
		log.Fatal("Failed to register Product entity:", err)
	}

	// Start the HTTP server
	fmt.Println("🚀 Development server starting with hot reload...")
	fmt.Println("Service endpoints:")
	fmt.Println("  Service Document: http://localhost:8080/")
	fmt.Println("  Metadata:        http://localhost:8080/$metadata")
	fmt.Println("  Products:        http://localhost:8080/Products")
	fmt.Println("  Single Product:  http://localhost:8080/Products(1)")
	fmt.Println()

	if err := service.ListenAndServe(":8080"); err != nil {
		log.Fatal("Server failed:", err)
	}
}
