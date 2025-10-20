package main

import "time"

// Address represents a complex type for physical addresses
type Address struct {
	Street     string `json:"Street" odata:"maxlength=100"`
	City       string `json:"City" odata:"maxlength=50,searchable"`
	State      string `json:"State" odata:"maxlength=2"`
	PostalCode string `json:"PostalCode" odata:"maxlength=10"`
	Country    string `json:"Country" odata:"maxlength=50"`
}

// Dimensions represents a complex type for product dimensions
type Dimensions struct {
	Length float64 `json:"Length" odata:"precision=10,scale=2"`
	Width  float64 `json:"Width" odata:"precision=10,scale=2"`
	Height float64 `json:"Height" odata:"precision=10,scale=2"`
	Unit   string  `json:"Unit" odata:"maxlength=10"` // e.g., "cm", "in"
}

// Category represents a product category entity
type Category struct {
	ID          uint   `json:"ID" gorm:"primaryKey" odata:"key"`
	Name        string `json:"Name" gorm:"not null;unique" odata:"required,maxlength=100"`
	Description string `json:"Description" odata:"maxlength=500"`
	// Navigation property for Products
	Products []Product `json:"Products,omitempty" gorm:"foreignKey:CategoryID;references:ID"`
}

// ProductStatus represents product status as a flags enum
type ProductStatus int

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

// Product represents a product entity for the development server with rich metadata
type Product struct {
	ID         uint          `json:"ID" gorm:"primaryKey" odata:"key"`
	Name       string        `json:"Name" gorm:"not null" odata:"required,maxlength=100,searchable"`
	Price      float64       `json:"Price" gorm:"not null" odata:"required,precision=10,scale=2"`
	CategoryID *uint         `json:"CategoryID" odata:"nullable"` // Foreign key for Category navigation property
	Status     ProductStatus `json:"Status" gorm:"not null" odata:"enum=ProductStatus,flags"`
	Version    int           `json:"Version" gorm:"default:1" odata:"etag"` // Version field used for optimistic concurrency control via ETag
	CreatedAt  time.Time     `json:"CreatedAt" gorm:"not null"`
	// Complex type properties
	ShippingAddress *Address    `json:"ShippingAddress,omitempty" gorm:"embedded;embeddedPrefix:shipping_" odata:"nullable"`
	Dimensions      *Dimensions `json:"Dimensions,omitempty" gorm:"embedded;embeddedPrefix:dim_" odata:"nullable"`
	// Navigation properties
	Category         *Category            `json:"Category,omitempty" gorm:"foreignKey:CategoryID;references:ID"`
	Descriptions     []ProductDescription `json:"Descriptions,omitempty" gorm:"foreignKey:ProductID;references:ID"`
	RelatedProducts  []Product            `json:"RelatedProducts,omitempty" gorm:"many2many:product_relations;"`
}

// ProductDescription represents a multilingual product description entity with rich metadata
type ProductDescription struct {
	ProductID   uint    `json:"ProductID" gorm:"primaryKey" odata:"key"`
	LanguageKey string  `json:"LanguageKey" gorm:"primaryKey;size:2" odata:"key,maxlength=2"`
	Description string  `json:"Description" gorm:"not null" odata:"required,maxlength=500,searchable"`
	LongText    *string `json:"LongText" gorm:"type:text" odata:"maxlength=2000,nullable,searchable"`
	// Navigation property back to Product
	Product *Product `json:"Product,omitempty" gorm:"foreignKey:ProductID;references:ID"`
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

// GetSampleCategories returns sample category data for seeding the database
func GetSampleCategories() []Category {
	return []Category{
		{
			ID:          1,
			Name:        "Electronics",
			Description: "Electronic devices and accessories",
		},
		{
			ID:          2,
			Name:        "Kitchen",
			Description: "Kitchen appliances and utensils",
		},
		{
			ID:          3,
			Name:        "Furniture",
			Description: "Home and office furniture",
		},
	}
}

// GetSampleProductDescriptions returns sample product description data for seeding the database
func GetSampleProductDescriptions() []ProductDescription {
	return []ProductDescription{
		{
			ProductID:   1,
			LanguageKey: "EN",
			Description: "High-performance laptop for productivity and gaming",
			LongText:    stringPtr("This state-of-the-art laptop features the latest processor technology, dedicated graphics card, and ample RAM to handle all your computing needs. Perfect for both professional work and entertainment."),
		},
		{
			ProductID:   1,
			LanguageKey: "DE",
			Description: "Hochleistungs-Laptop für Produktivität und Gaming",
			LongText:    stringPtr("Dieser hochmoderne Laptop verfügt über die neueste Prozessortechnologie, eine dedizierte Grafikkarte und reichlich RAM, um alle Ihre Computeranforderungen zu erfüllen. Perfekt für berufliche Arbeit und Unterhaltung."),
		},
		{
			ProductID:   2,
			LanguageKey: "EN",
			Description: "Ergonomic wireless mouse with precision tracking",
			LongText:    stringPtr("Experience comfort and precision with this wireless mouse. Features adjustable DPI settings, long battery life, and a contoured design that fits perfectly in your hand."),
		},
		{
			ProductID:   2,
			LanguageKey: "FR",
			Description: "Souris sans fil ergonomique avec suivi de précision",
			LongText:    stringPtr("Découvrez le confort et la précision avec cette souris sans fil. Dispose de paramètres DPI réglables, d'une longue durée de vie de la batterie et d'un design profilé qui s'adapte parfaitement à votre main."),
		},
		{
			ProductID:   3,
			LanguageKey: "EN",
			Description: "Ceramic coffee mug with heat retention technology",
			LongText:    stringPtr("Keep your beverages at the perfect temperature with this innovative ceramic mug. The double-wall construction provides excellent insulation while remaining comfortable to hold."),
		},
		{
			ProductID:   5,
			LanguageKey: "EN",
			Description: "Latest generation smartphone with advanced camera",
			LongText:    stringPtr("Capture life's moments in stunning detail with our flagship smartphone. Features a professional-grade camera system, lightning-fast processor, and all-day battery life."),
		},
		{
			ProductID:   5,
			LanguageKey: "ES",
			Description: "Smartphone de última generación con cámara avanzada",
			LongText:    stringPtr("Captura los momentos de la vida con un detalle asombroso con nuestro smartphone insignia. Cuenta con un sistema de cámara de nivel profesional, procesador ultrarrápido y batería para todo el día."), //nolint:misspell // Spanish text
		},
	}
}

// stringPtr is a helper function to create a pointer to a string
func stringPtr(s string) *string {
	return &s
}

// CompanyInfo represents a singleton entity for company information
// This demonstrates the singleton feature - a single entity accessed directly by name
type CompanyInfo struct {
	ID          uint      `json:"ID" gorm:"primaryKey" odata:"key"`
	Name        string    `json:"Name" gorm:"not null" odata:"required,maxlength=200"`
	CEO         string    `json:"CEO" gorm:"not null" odata:"required,maxlength=100"`
	Founded     int       `json:"Founded" gorm:"not null"`
	HeadQuarter string    `json:"HeadQuarter" gorm:"not null" odata:"maxlength=200"`
	Website     string    `json:"Website" gorm:"not null" odata:"maxlength=100"`
	Logo        []byte    `json:"Logo" gorm:"type:blob" odata:"nullable,contenttype=image/svg+xml"` // Binary data example (company logo)
	Version     int       `json:"Version" gorm:"default:1" odata:"etag"`
	UpdatedAt   time.Time `json:"UpdatedAt" gorm:"not null"`
}

// GetCompanyInfo returns the singleton company information
func GetCompanyInfo() CompanyInfo {
	// Create a simple SVG logo as binary data
	svgLogo := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100" viewBox="0 0 100 100">
  <rect width="100" height="100" fill="#4A90E2"/>
  <text x="50" y="55" font-family="Arial" font-size="24" fill="white" text-anchor="middle">TS</text>
</svg>`)

	return CompanyInfo{
		ID:          1,
		Name:        "TechStore Inc.",
		CEO:         "Sarah Johnson",
		Founded:     2010,
		HeadQuarter: "Seattle, WA, USA",
		Website:     "https://techstore.example.com",
		Logo:        svgLogo,
		Version:     1,
		UpdatedAt:   time.Now(),
	}
}
