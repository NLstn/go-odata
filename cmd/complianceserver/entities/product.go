package entities

import (
	"time"
)

// ProductStatus represents product status as a flags enum
type ProductStatus int32

const (
	// ProductStatusNone represents no status
	ProductStatusNone ProductStatus = 0
	// ProductStatusInStock represents that the product is in stock
	ProductStatusInStock ProductStatus = 1
	// ProductStatusOnSale represents that the product is on sale
	ProductStatusOnSale ProductStatus = 2
	// ProductStatusDiscontinued represents that the product is discontinued
	ProductStatusDiscontinued ProductStatus = 4
	// ProductStatusFeatured represents that the product is featured
	ProductStatusFeatured ProductStatus = 8
)

// EnumMembers returns the enum member mapping for compliance metadata generation.
func (ProductStatus) EnumMembers() map[string]int {
	return map[string]int{
		"None":         int(ProductStatusNone),
		"InStock":      int(ProductStatusInStock),
		"OnSale":       int(ProductStatusOnSale),
		"Discontinued": int(ProductStatusDiscontinued),
		"Featured":     int(ProductStatusFeatured),
	}
}

// Product represents a product entity for the compliance server
type Product struct {
	ID          uint          `json:"ID" gorm:"primaryKey" odata:"key"`
	Name        string        `json:"Name" gorm:"not null" odata:"required,maxlength=100,searchable"`
	Description *string       `json:"Description" odata:"nullable,maxlength=500"` // Nullable description field
	Price       float64       `json:"Price" gorm:"not null" odata:"required,precision=10,scale=2"`
	CategoryID  *uint         `json:"CategoryID" odata:"nullable"` // Foreign key for Category navigation property
	Status      ProductStatus `json:"Status" gorm:"not null" odata:"enum=ProductStatus,flags"`
	Version     int           `json:"Version" gorm:"default:1" odata:"etag"` // Version field used for optimistic concurrency control via ETag
	CreatedAt   time.Time     `json:"CreatedAt" gorm:"not null"`
	// Complex type properties
	ShippingAddress *Address    `json:"ShippingAddress,omitempty" gorm:"embedded;embeddedPrefix:shipping_" odata:"nullable"`
	Dimensions      *Dimensions `json:"Dimensions,omitempty" gorm:"embedded;embeddedPrefix:dim_" odata:"nullable"`
	// Navigation properties
	Category        *Category            `json:"Category,omitempty" gorm:"foreignKey:CategoryID;references:ID"`
	Descriptions    []ProductDescription `json:"Descriptions,omitempty" gorm:"foreignKey:ProductID;references:ID"`
	RelatedProducts []Product            `json:"RelatedProducts,omitempty" gorm:"many2many:product_relations;"`
}

// GetSampleProducts returns sample product data for seeding the database
func GetSampleProducts() []Product {
	categoryElectronics := uint(1)
	categoryKitchen := uint(2)
	categoryFurniture := uint(3)

	return []Product{
		{
			ID:         1,
			Name:       "Laptop",
			Price:      999.99,
			CategoryID: &categoryElectronics,
			Status:     ProductStatusInStock | ProductStatusFeatured, // In stock and featured
			Version:    1,
			CreatedAt:  time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			ShippingAddress: &Address{
				Street:     "123 Tech Way",
				City:       "Seattle",
				State:      "WA",
				PostalCode: "98101",
				Country:    "USA",
			},
			Dimensions: &Dimensions{
				Length: 35.5,
				Width:  25.0,
				Height: 2.5,
				Unit:   "cm",
			},
		},
		{
			ID:         2,
			Name:       "Wireless Mouse",
			Price:      29.99,
			CategoryID: &categoryElectronics,
			Status:     ProductStatusInStock | ProductStatusOnSale, // In stock and on sale
			Version:    1,
			CreatedAt:  time.Date(2024, 3, 20, 14, 45, 0, 0, time.UTC),
			ShippingAddress: &Address{
				Street:     "456 Innovation Blvd",
				City:       "San Francisco",
				State:      "CA",
				PostalCode: "94102",
				Country:    "USA",
			},
			Dimensions: &Dimensions{
				Length: 10.0,
				Width:  6.0,
				Height: 4.0,
				Unit:   "cm",
			},
		},
		{
			ID:         3,
			Name:       "Coffee Mug",
			Price:      15.50,
			CategoryID: &categoryKitchen,
			Status:     ProductStatusInStock, // Only in stock
			Version:    1,
			CreatedAt:  time.Date(2023, 11, 5, 9, 15, 0, 0, time.UTC),
			ShippingAddress: &Address{
				Street:     "789 Home St",
				City:       "Portland",
				State:      "OR",
				PostalCode: "97201",
				Country:    "USA",
			},
			Dimensions: &Dimensions{
				Length: 8.0,
				Width:  8.0,
				Height: 10.0,
				Unit:   "cm",
			},
		},
		{
			ID:         4,
			Name:       "Office Chair",
			Price:      249.99,
			CategoryID: &categoryFurniture,
			Status:     ProductStatusDiscontinued, // Discontinued
			Version:    1,
			CreatedAt:  time.Date(2023, 8, 12, 16, 20, 0, 0, time.UTC),
			// No shipping address or dimensions (testing null complex types)
			ShippingAddress: nil,
			Dimensions:      nil,
		},
		{
			ID:         5,
			Name:       "Smartphone",
			Price:      799.99,
			CategoryID: &categoryElectronics,
			Status:     ProductStatusInStock | ProductStatusOnSale | ProductStatusFeatured, // In stock, on sale, and featured
			Version:    1,
			CreatedAt:  time.Date(2024, 6, 28, 11, 0, 0, 0, time.UTC),
			ShippingAddress: &Address{
				Street:     "321 Mobile Ave",
				City:       "Austin",
				State:      "TX",
				PostalCode: "78701",
				Country:    "USA",
			},
			Dimensions: &Dimensions{
				Length: 15.0,
				Width:  7.5,
				Height: 0.8,
				Unit:   "cm",
			},
		},
	}
}
