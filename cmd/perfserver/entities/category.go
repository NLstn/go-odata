package entities

// Category represents a product category entity
type Category struct {
	ID          uint   `json:"ID" gorm:"primaryKey" odata:"key"`
	Name        string `json:"Name" gorm:"not null;unique" odata:"required,maxlength=100"`
	Description string `json:"Description" odata:"maxlength=500"`
	// Navigation property for Products
	Products []Product `json:"Products,omitempty" gorm:"foreignKey:CategoryID;references:ID"`
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
