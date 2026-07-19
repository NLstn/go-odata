package entities

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	odata "github.com/nlstn/go-odata"
	"gorm.io/gorm"
)

// ReleaseDateValue is a string-backed named type declared as Edm.Date via an
// explicit UnderlyingType override (Go has no distinct native type for Edm.Date).
// Holds ISO-8601 date strings, e.g. "2024-01-15".
type ReleaseDateValue string

// TimeOfDayValue is a string-backed named type declared as Edm.TimeOfDay via an
// explicit UnderlyingType override. Holds ISO-8601 time-of-day strings, e.g. "09:30:00".
type TimeOfDayValue string

// DurationValue is a string-backed named type declared as Edm.Duration via an
// explicit UnderlyingType override. Holds ISO-8601 duration strings, e.g. "P1D", "PT45S".
type DurationValue string

func init() {
	if err := odata.RegisterTypeDefinition(ReleaseDateValue(""), "ReleaseDateValue", odata.TypeDefinitionFacets{UnderlyingType: "Edm.Date"}); err != nil {
		panic("failed to register ReleaseDateValue TypeDefinition: " + err.Error())
	}
	if err := odata.RegisterTypeDefinition(TimeOfDayValue(""), "TimeOfDayValue", odata.TypeDefinitionFacets{UnderlyingType: "Edm.TimeOfDay"}); err != nil {
		panic("failed to register TimeOfDayValue TypeDefinition: " + err.Error())
	}
	if err := odata.RegisterTypeDefinition(DurationValue(""), "DurationValue", odata.TypeDefinitionFacets{UnderlyingType: "Edm.Duration"}); err != nil {
		panic("failed to register DurationValue TypeDefinition: " + err.Error())
	}
}

func releaseDatePtr(s string) *ReleaseDateValue {
	v := ReleaseDateValue(s)
	return &v
}

func timeOfDayPtr(s string) *TimeOfDayValue {
	v := TimeOfDayValue(s)
	return &v
}

func durationPtr(s string) *DurationValue {
	v := DurationValue(s)
	return &v
}

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

// RatingValue wraps an unsigned 8-bit rating while accepting wider SQL Server scan values.
type RatingValue uint8

// TemperatureValue wraps a signed 8-bit temperature while accepting wider SQL Server scan values.
type TemperatureValue int8

// QuantityValue wraps a signed 16-bit quantity while accepting wider SQL Server scan values.
type QuantityValue int16

func scanInt64Value(value any) (int64, error) {
	switch v := value.(type) {
	case int64:
		return v, nil
	case int32:
		return int64(v), nil
	case int16:
		return int64(v), nil
	case int8:
		return int64(v), nil
	case uint64:
		if v > math.MaxInt64 {
			return 0, fmt.Errorf("value %d out of range for signed integer", v)
		}
		return int64(v), nil
	case uint32:
		return int64(v), nil
	case uint16:
		return int64(v), nil
	case uint8:
		return int64(v), nil
	case []byte:
		n, err := strconv.ParseInt(string(v), 10, 64)
		if err != nil {
			return 0, err
		}
		return n, nil
	case string:
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, err
		}
		return n, nil
	default:
		return 0, fmt.Errorf("cannot scan %T into signed integer", value)
	}
}

func scanUint64Value(value any) (uint64, error) {
	switch v := value.(type) {
	case uint64:
		return v, nil
	case uint32:
		return uint64(v), nil
	case uint16:
		return uint64(v), nil
	case uint8:
		return uint64(v), nil
	case int64:
		if v < 0 {
			return 0, fmt.Errorf("value %d out of range for unsigned integer", v)
		}
		return uint64(v), nil
	case int32:
		if v < 0 {
			return 0, fmt.Errorf("value %d out of range for unsigned integer", v)
		}
		return uint64(v), nil
	case int16:
		if v < 0 {
			return 0, fmt.Errorf("value %d out of range for unsigned integer", v)
		}
		return uint64(v), nil
	case int8:
		if v < 0 {
			return 0, fmt.Errorf("value %d out of range for unsigned integer", v)
		}
		return uint64(v), nil
	case []byte:
		n, err := strconv.ParseUint(string(v), 10, 64)
		if err != nil {
			return 0, err
		}
		return n, nil
	case string:
		n, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return 0, err
		}
		return n, nil
	default:
		return 0, fmt.Errorf("cannot scan %T into unsigned integer", value)
	}
}

func (v *RatingValue) Scan(value any) error {
	if value == nil {
		*v = 0
		return nil
	}

	n, err := scanUint64Value(value)
	if err != nil {
		return err
	}
	if n > math.MaxUint8 {
		return fmt.Errorf("value %d out of range for RatingValue", n)
	}
	*v = RatingValue(n)
	return nil
}

func (v RatingValue) Value() (driver.Value, error) {
	return int64(v), nil
}

func (v *TemperatureValue) Scan(value any) error {
	if value == nil {
		*v = 0
		return nil
	}

	n, err := scanInt64Value(value)
	if err != nil {
		return err
	}
	if n < math.MinInt8 || n > math.MaxInt8 {
		return fmt.Errorf("value %d out of range for TemperatureValue", n)
	}
	*v = TemperatureValue(n)
	return nil
}

func (v TemperatureValue) Value() (driver.Value, error) {
	return int64(v), nil
}

func (v *QuantityValue) Scan(value any) error {
	if value == nil {
		*v = 0
		return nil
	}

	n, err := scanInt64Value(value)
	if err != nil {
		return err
	}
	if n < math.MinInt16 || n > math.MaxInt16 {
		return fmt.Errorf("value %d out of range for QuantityValue", n)
	}
	*v = QuantityValue(n)
	return nil
}

func (v QuantityValue) Value() (driver.Value, error) {
	return int64(v), nil
}

// UnmarshalJSON decodes OData enum strings (e.g. "InStock,Featured") back to ProductStatus.
func (s *ProductStatus) UnmarshalJSON(data []byte) error {
	var n int32
	if err := json.Unmarshal(data, &n); err == nil {
		*s = ProductStatus(n)
		return nil
	}
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	members := map[string]ProductStatus{
		"None":         ProductStatusNone,
		"InStock":      ProductStatusInStock,
		"OnSale":       ProductStatusOnSale,
		"Discontinued": ProductStatusDiscontinued,
		"Featured":     ProductStatusFeatured,
	}
	var result ProductStatus
	for _, part := range strings.Split(str, ",") {
		v, ok := members[strings.TrimSpace(part)]
		if !ok {
			return fmt.Errorf("unknown ProductStatus member: %q", strings.TrimSpace(part))
		}
		result |= v
	}
	*s = result
	return nil
}

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
	ID              uuid.UUID         `json:"ID" gorm:"type:char(36);primaryKey" odata:"key,generate=uuid"`
	Name            string            `json:"Name" gorm:"not null" odata:"required,maxlength=100,searchable,annotation:Core.Description=Product display name"`
	Description     *string           `json:"Description" odata:"nullable,maxlength=500,annotation:Core.Description=Detailed product description"` // Nullable description field
	Price           float64           `json:"Price" gorm:"not null" odata:"required,precision=10,scale=2"`
	Rating          RatingValue       `json:"Rating" odata:""`
	Temperature     TemperatureValue  `json:"Temperature" odata:""`
	Quantity        QuantityValue     `json:"Quantity" odata:""`
	Weight          float32           `json:"Weight" odata:""`
	Data            []byte            `json:"Data,omitempty" odata:"nullable"`
	ReleaseDate     *ReleaseDateValue `json:"ReleaseDate,omitempty" odata:"nullable"`
	OpenTime        *TimeOfDayValue   `json:"OpenTime,omitempty" odata:"nullable"`
	ShippingTime    *DurationValue    `json:"ShippingTime,omitempty" odata:"nullable"`
	ProcessingTime  *DurationValue    `json:"ProcessingTime,omitempty" odata:"nullable"`
	Offset          *DurationValue    `json:"Offset,omitempty" odata:"nullable"`
	CategoryID      *uuid.UUID        `json:"CategoryID" gorm:"type:char(36)" odata:"nullable"` // Foreign key for Category navigation property
	Status          ProductStatus     `json:"Status" gorm:"not null" odata:"enum=ProductStatus,flags"`
	Version         int               `json:"Version" gorm:"default:1" odata:"etag"` // Version field used for optimistic concurrency control via ETag
	CreatedAt       time.Time         `json:"CreatedAt" gorm:"not null" odata:"annotation:Core.Computed"`
	SerialNumber    *string           `json:"SerialNumber,omitempty" gorm:"type:varchar(50)" odata:"nullable,maxlength=50,annotation:Core.Immutable,annotation:Core.Description=Unique serial number assigned at creation"`
	ProductType     string            `json:"ProductType,omitempty" gorm:"default:'Product'" odata:"maxlength=50"` // Discriminator for type inheritance
	SpecialProperty *string           `json:"SpecialProperty,omitempty" odata:"nullable,maxlength=200"`            // Property for SpecialProduct derived type
	SpecialFeature  *string           `json:"SpecialFeature,omitempty" odata:"nullable,maxlength=100"`             // Property for SpecialProduct derived type
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
	Descriptions    []ProductDescription `json:"Descriptions,omitempty" gorm:"foreignKey:ProductID;references:ID;constraint:OnDelete:CASCADE"`
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
			Rating:          200,
			Temperature:     -10,
			Quantity:        1200,
			Weight:          3.14,
			Data:            []byte("test"),
			ReleaseDate:     releaseDatePtr("2024-01-15"),
			OpenTime:        timeOfDayPtr("09:30:00"),
			ShippingTime:    durationPtr("P1D"),
			ProcessingTime:  durationPtr("PT45S"),
			Offset:          durationPtr("P0D"),
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
			Name:           "Wireless Mouse",
			Price:          29.99,
			Rating:         110,
			Temperature:    5,
			Quantity:       300,
			Weight:         0.09,
			Data:           []byte{0x01, 0x02, 0x03},
			ReleaseDate:    releaseDatePtr("2024-01-20"),
			OpenTime:       timeOfDayPtr("08:15:00"),
			ShippingTime:   durationPtr("PT2H"),
			ProcessingTime: durationPtr("PT1.5S"),
			Offset:         durationPtr("-P1D"),
			CategoryID:     nil,                                        // Will be set during seeding
			Status:         ProductStatusInStock | ProductStatusOnSale, // In stock and on sale
			Version:        1,
			CreatedAt:      time.Date(2024, 3, 20, 14, 45, 0, 0, time.UTC),
			ProductType:    "Product",
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
			Name:           "Coffee Mug",
			Price:          15.50,
			Rating:         0,
			Temperature:    -5,
			Quantity:       -200,
			Weight:         0.0,
			Data:           []byte{},
			ReleaseDate:    releaseDatePtr("2024-01-01"),
			OpenTime:       timeOfDayPtr("00:00:00"),
			ShippingTime:   durationPtr("P2D"),
			ProcessingTime: durationPtr("PT30M"),
			Offset:         durationPtr("P0D"),
			CategoryID:     nil,                  // Will be set during seeding
			Status:         ProductStatusInStock, // Only in stock
			Version:        1,
			CreatedAt:      time.Date(2023, 11, 5, 9, 15, 0, 0, time.UTC),
			ProductType:    "Product",
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
			Name:           "Office Chair",
			Price:          249.99,
			Rating:         255,
			Temperature:    127,
			Quantity:       32767,
			Weight:         150.0,
			ReleaseDate:    releaseDatePtr("2024-12-31"),
			OpenTime:       timeOfDayPtr("23:59:59"),
			ShippingTime:   durationPtr("P1DT2H30M"),
			ProcessingTime: durationPtr("PT45S"),
			Offset:         durationPtr("P0D"),
			CategoryID:     nil,                       // Will be set during seeding
			Status:         ProductStatusDiscontinued, // Discontinued
			Version:        1,
			CreatedAt:      time.Date(2023, 8, 12, 16, 20, 0, 0, time.UTC),
			ProductType:    "Product",
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
