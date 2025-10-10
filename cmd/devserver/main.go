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

	// Auto-migrate the Product and ProductDescription models
	if err := db.AutoMigrate(&Product{}, &ProductDescription{}); err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	// Seed the database with sample data
	sampleProducts := GetSampleProducts()
	if err := db.Create(&sampleProducts).Error; err != nil {
		log.Fatal("Failed to seed database:", err)
	}

	sampleDescriptions := GetSampleProductDescriptions()
	if err := db.Create(&sampleDescriptions).Error; err != nil {
		log.Fatal("Failed to seed product descriptions:", err)
	}

	fmt.Printf("Database initialized with %d products and %d descriptions\n", len(sampleProducts), len(sampleDescriptions))

	// Create OData service
	service := odata.NewService(db)

	// Register the Product and ProductDescription entities
	if err := service.RegisterEntity(&Product{}); err != nil {
		log.Fatal("Failed to register Product entity:", err)
	}
	if err := service.RegisterEntity(&ProductDescription{}); err != nil {
		log.Fatal("Failed to register ProductDescription entity:", err)
	}

	// Start the HTTP server
	fmt.Println("🚀 Development server starting with hot reload...")
	fmt.Println("Service endpoints:")
	fmt.Println("  Service Document:     http://localhost:8080/")
	fmt.Println("  Metadata (XML):       http://localhost:8080/$metadata")
	fmt.Println("  Metadata (JSON):      http://localhost:8080/$metadata?$format=json")
	fmt.Println("  Products:             http://localhost:8080/Products")
	fmt.Println("  Single Product:       http://localhost:8080/Products(1)")
	fmt.Println("  ProductDescriptions:  http://localhost:8080/ProductDescriptions")
	fmt.Println("  Product Descriptions: http://localhost:8080/ProductDescriptions(ProductID=1,LanguageKey='EN')")
	fmt.Println()

	if err := service.ListenAndServe(":8080"); err != nil {
		log.Fatal("Server failed:", err)
	}
}
