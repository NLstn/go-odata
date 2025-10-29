package entities

// SpecialProduct is a derived type that extends Product
// This is used for testing type casting and inheritance features
type SpecialProduct struct {
	Product                 // Embedded Product - all Product fields are inherited
	SpecialProperty string  `json:"SpecialProperty" odata:"maxlength=200"`
	SpecialFeature  *string `json:"SpecialFeature,omitempty" odata:"nullable,maxlength=100"`
}

// GetSampleSpecialProducts returns sample special product data for seeding the database
func GetSampleSpecialProducts() []SpecialProduct {
	categoryElectronics := uint(1)

	desc := "Premium feature"
	return []SpecialProduct{
		{
			Product: Product{
				ID:         10,
				Name:       "Premium Laptop",
				Price:      1999.99,
				CategoryID: &categoryElectronics,
				Status:     ProductStatusInStock | ProductStatusFeatured,
				Version:    1,
			},
			SpecialProperty: "Extra warranty included",
			SpecialFeature:  &desc,
		},
		{
			Product: Product{
				ID:         11,
				Name:       "Gaming Mouse Pro",
				Price:      79.99,
				CategoryID: &categoryElectronics,
				Status:     ProductStatusInStock | ProductStatusOnSale,
				Version:    1,
			},
			SpecialProperty: "RGB lighting",
			SpecialFeature:  nil,
		},
	}
}
