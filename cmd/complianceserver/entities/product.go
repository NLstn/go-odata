package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
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
	ID              uuid.UUID     `json:"ID" gorm:"type:char(36);primaryKey" odata:"key,generate=uuid"`
	Name            string        `json:"Name" gorm:"not null" odata:"required,maxlength=100,searchable,annotation:Core.Description=Product display name"`
	Description     *string       `json:"Description" odata:"nullable,maxlength=500,annotation:Core.Description=Detailed product description"` // Nullable description field
	Price           float64       `json:"Price" gorm:"not null" odata:"required,precision=10,scale=2"`
	CategoryID      *uuid.UUID    `json:"CategoryID" gorm:"type:char(36)" odata:"nullable"` // Foreign key for Category navigation property
	Status          ProductStatus `json:"Status" gorm:"not null" odata:"enum=ProductStatus,flags"`
	Version         int           `json:"Version" gorm:"default:1" odata:"etag"` // Version field used for optimistic concurrency control via ETag
	CreatedAt       time.Time     `json:"CreatedAt" gorm:"not null" odata:"annotation:Core.Computed"`
	SerialNumber    *string       `json:"SerialNumber,omitempty" gorm:"type:varchar(50)" odata:"nullable,maxlength=50,annotation:Core.Immutable,annotation:Core.Description=Unique serial number assigned at creation"`
	ProductType     string        `json:"ProductType,omitempty" gorm:"default:'Product'" odata:"maxlength=50"` // Discriminator for type inheritance
	SpecialProperty *string       `json:"SpecialProperty,omitempty" odata:"nullable,maxlength=200"`            // Property for SpecialProduct derived type
	SpecialFeature  *string       `json:"SpecialFeature,omitempty" odata:"nullable,maxlength=100"`             // Property for SpecialProduct derived type
	// Complex type properties
	ShippingAddress *Address    `json:"ShippingAddress,omitempty" gorm:"embedded;embeddedPrefix:shipping_" odata:"nullable"`
	Dimensions      *Dimensions `json:"Dimensions,omitempty" gorm:"embedded;embeddedPrefix:dim_" odata:"nullable"`
	// Geospatial properties (stored as WKT strings for compatibility)
	Location *string `json:"Location,omitempty" odata:"nullable"` // Geography Point in WKT format
	Route    *string `json:"Route,omitempty" odata:"nullable"`    // Geography LineString in WKT format
	Area     *string `json:"Area,omitempty" odata:"nullable"`     // Geography Polygon in WKT format
	// Stream properties
	Photo            struct{} `json:"-" gorm:"-" odata:"stream"`  // Photo stream property (logical property, no storage)
	PhotoContentType string   `json:"-" gorm:"type:varchar(100)"` // Content type for Photo stream
	PhotoContent     []byte   `json:"-"`                          // Photo stream content
	// Navigation properties
	Category        *Category            `json:"Category,omitempty" gorm:"foreignKey:CategoryID;references:ID"`
	Descriptions    []ProductDescription `json:"Descriptions,omitempty" gorm:"foreignKey:ProductID;references:ID"`
	RelatedProducts []Product            `json:"RelatedProducts,omitempty" gorm:"many2many:product_relations;"`
}

// TableName overrides the table name used by GORM to match OData entity set name
func (Product) TableName() string {
	return "Products"
}

// BeforeUpdate is a GORM hook that increments the Version field before each update
// This ensures the ETag changes whenever the entity is modified
func (p *Product) BeforeUpdate(tx *gorm.DB) error {
	p.Version++
	return nil
}

// GetStreamProperty returns the content of a stream property by name
func (p *Product) GetStreamProperty(name string) ([]byte, string, bool) {
	if name == "Photo" {
		return p.PhotoContent, p.PhotoContentType, true
	}
	return nil, "", false
}

// SetStreamProperty sets the content of a stream property by name
func (p *Product) SetStreamProperty(name string, content []byte, contentType string) bool {
	if name == "Photo" {
		p.PhotoContent = content
		p.PhotoContentType = contentType
		return true
	}
	return false
}

// GetSampleProducts returns sample product data for seeding the database
// Note: IDs are server-generated and should not be set in sample data
func GetSampleProducts() []Product {
	return []Product{
		// Special Product - used by type casting tests
		// Kept as "Laptop" name for backward compatibility with primitive type tests
		{
			Name:            "Laptop",
			Price:           999.99,
			CategoryID:      nil, // Will be set during seeding after categories are created
			Status:          ProductStatusInStock | ProductStatusFeatured,
			Version:         1,
			CreatedAt:       time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			ProductType:     "SpecialProduct",
			SpecialProperty: stringPtr("Extended warranty included"),
			SpecialFeature:  stringPtr("Premium support"),
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
			Location: stringPtr("POINT(-122.3321 47.6062)"),                                                   // Seattle
			Route:    stringPtr("LINESTRING(-122.3321 47.6062, -122.4194 47.2529)"),                           // Seattle to Tacoma
			Area:     stringPtr("POLYGON((-122.5 47.5, -122.0 47.5, -122.0 47.8, -122.5 47.8, -122.5 47.5))"), // Seattle area
		},
		{
			Name:        "Wireless Mouse",
			Price:       29.99,
			CategoryID:  nil,                                        // Will be set during seeding
			Status:      ProductStatusInStock | ProductStatusOnSale, // In stock and on sale
			Version:     1,
			CreatedAt:   time.Date(2024, 3, 20, 14, 45, 0, 0, time.UTC),
			ProductType: "Product",
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
			Location: stringPtr("POINT(-122.4194 37.7749)"),                                                   // San Francisco
			Route:    stringPtr("LINESTRING(-122.4194 37.7749, -122.2711 37.8044)"),                           // SF to Oakland
			Area:     stringPtr("POLYGON((-122.5 37.7, -122.3 37.7, -122.3 37.9, -122.5 37.9, -122.5 37.7))"), // SF Bay area
		},
		{
			Name:        "Coffee Mug",
			Price:       15.50,
			CategoryID:  nil,                  // Will be set during seeding
			Status:      ProductStatusInStock, // Only in stock
			Version:     1,
			CreatedAt:   time.Date(2023, 11, 5, 9, 15, 0, 0, time.UTC),
			ProductType: "Product",
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
			Location: stringPtr("POINT(-122.6765 45.5231)"),                                                   // Portland
			Route:    stringPtr("LINESTRING(-122.6765 45.5231, -122.6587 45.5152)"),                           // Portland downtown
			Area:     stringPtr("POLYGON((-122.8 45.4, -122.5 45.4, -122.5 45.7, -122.8 45.7, -122.8 45.4))"), // Portland area
		},
		{
			Name:        "Office Chair",
			Price:       249.99,
			CategoryID:  nil,                       // Will be set during seeding
			Status:      ProductStatusDiscontinued, // Discontinued
			Version:     1,
			CreatedAt:   time.Date(2023, 8, 12, 16, 20, 0, 0, time.UTC),
			ProductType: "Product",
			// No shipping address or dimensions (testing null complex types)
			ShippingAddress: nil,
			Dimensions:      nil,
		},
		{
			Name:        "Smartphone",
			Price:       799.99,
			CategoryID:  nil,                                                                // Will be set during seeding
			Status:      ProductStatusInStock | ProductStatusOnSale | ProductStatusFeatured, // In stock, on sale, and featured
			Version:     1,
			CreatedAt:   time.Date(2024, 6, 28, 11, 0, 0, 0, time.UTC),
			ProductType: "Product",
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
			Location: stringPtr("POINT(-97.7431 30.2672)"),                                               // Austin
			Route:    stringPtr("LINESTRING(-97.7431 30.2672, -97.7426 30.2849)"),                        // Austin downtown
			Area:     stringPtr("POLYGON((-98.0 30.1, -97.5 30.1, -97.5 30.5, -98.0 30.5, -98.0 30.1))"), // Austin area
		},
		// Special Products (derived type)
		{
			Name:            "Premium Laptop Pro",
			Price:           1999.99,
			CategoryID:      nil, // Will be set during seeding
			Status:          ProductStatusInStock | ProductStatusFeatured,
			Version:         1,
			CreatedAt:       time.Date(2024, 7, 1, 10, 0, 0, 0, time.UTC),
			ProductType:     "SpecialProduct",
			SpecialProperty: stringPtr("Extended 5-year warranty"),
			SpecialFeature:  stringPtr("Premium support package"),
			ShippingAddress: &Address{
				Street:     "999 Premium Way",
				City:       "New York",
				State:      "NY",
				PostalCode: "10001",
				Country:    "USA",
			},
			Dimensions: &Dimensions{
				Length: 40.0,
				Width:  28.0,
				Height: 2.0,
				Unit:   "cm",
			},
			Location: stringPtr("POINT(-74.0060 40.7128)"),                                               // New York City
			Route:    stringPtr("LINESTRING(-74.0060 40.7128, -73.9352 40.7306)"),                        // Manhattan to Queens
			Area:     stringPtr("POLYGON((-74.1 40.6, -73.8 40.6, -73.8 40.9, -74.1 40.9, -74.1 40.6))"), // NYC area
		},
		{
			Name:            "Gaming Mouse Ultra",
			Price:           149.99,
			CategoryID:      nil, // Will be set during seeding
			Status:          ProductStatusInStock | ProductStatusOnSale,
			Version:         1,
			CreatedAt:       time.Date(2024, 7, 15, 14, 30, 0, 0, time.UTC),
			ProductType:     "SpecialProduct",
			SpecialProperty: stringPtr("RGB lighting with 16 million colors"),
			SpecialFeature:  stringPtr("Customizable DPI settings"),
			ShippingAddress: &Address{
				Street:     "888 Gaming Blvd",
				City:       "Los Angeles",
				State:      "CA",
				PostalCode: "90001",
				Country:    "USA",
			},
			Dimensions: &Dimensions{
				Length: 12.5,
				Width:  7.5,
				Height: 4.5,
				Unit:   "cm",
			},
			Location: stringPtr("POINT(-118.2437 34.0522)"),                                                   // Los Angeles
			Route:    stringPtr("LINESTRING(-118.2437 34.0522, -118.3687 34.1016)"),                           // LA to Beverly Hills
			Area:     stringPtr("POLYGON((-118.5 33.9, -118.0 33.9, -118.0 34.3, -118.5 34.3, -118.5 33.9))"), // LA area
		},
	}
}
