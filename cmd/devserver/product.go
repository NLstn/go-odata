package main

// Product represents a product entity for the development server
type Product struct {
	ID          uint    `json:"ID" gorm:"primaryKey" odata:"key"`
	Name        string  `json:"Name" gorm:"not null"`
	Description string  `json:"Description"`
	Price       float64 `json:"Price" gorm:"not null"`
	Category    string  `json:"Category" gorm:"not null"`
}

// GetSampleProducts returns sample product data for seeding the database
func GetSampleProducts() []Product {
	return []Product{
		{
			ID:          1,
			Name:        "Laptop",
			Description: "High-performance laptop for productivity and gaming",
			Price:       999.99,
			Category:    "Electronics",
		},
		{
			ID:          2,
			Name:        "Wireless Mouse",
			Description: "Ergonomic wireless mouse with precision tracking",
			Price:       29.99,
			Category:    "Electronics",
		},
		{
			ID:          3,
			Name:        "Coffee Mug",
			Description: "Ceramic coffee mug with heat retention technology",
			Price:       15.50,
			Category:    "Kitchen",
		},
		{
			ID:          4,
			Name:        "Office Chair",
			Description: "Ergonomic office chair with lumbar support",
			Price:       249.99,
			Category:    "Furniture",
		},
		{
			ID:          5,
			Name:        "Smartphone",
			Description: "Latest generation smartphone with advanced camera",
			Price:       799.99,
			Category:    "Electronics",
		},
	}
}
