package entities

import "github.com/google/uuid"

// Category represents a product category entity
type Category struct {
	ID          uuid.UUID `json:"ID" gorm:"type:uuid;primaryKey" odata:"key,generate=uuid"`
	Name        string    `json:"Name" gorm:"not null;unique" odata:"required,maxlength=100"`
	Description string    `json:"Description" odata:"maxlength=500"`
	// Navigation property for Products
	Products []Product `json:"Products,omitempty" gorm:"foreignKey:CategoryID;references:ID"`
}

// TableName overrides the table name used by GORM to match OData entity set name
func (Category) TableName() string {
	return "Categories"
}

// GetSampleCategories returns sample category data for seeding the database
func GetSampleCategories() []Category {
	return []Category{
		{
			Name:        "Electronics",
			Description: "Electronic devices and accessories",
		},
		{
			Name:        "Kitchen",
			Description: "Kitchen appliances and utensils",
		},
		{
			Name:        "Furniture",
			Description: "Home and office furniture",
		},
	}
}
