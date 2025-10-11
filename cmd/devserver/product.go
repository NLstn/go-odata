package main

// Product represents a product entity for the development server with rich metadata
type Product struct {
	ID       uint    `json:"ID" gorm:"primaryKey" odata:"key"`
	Name     string  `json:"Name" gorm:"not null" odata:"required,maxlength=100"`
	Price    float64 `json:"Price" gorm:"not null" odata:"required,precision=10,scale=2"`
	Category string  `json:"Category" gorm:"not null" odata:"required,maxlength=50"`
	Version  int     `json:"Version" gorm:"default:1" odata:"etag"` // Version field used for optimistic concurrency control via ETag
	// Navigation property for ProductDescriptions
	Descriptions []ProductDescription `json:"Descriptions" gorm:"foreignKey:ProductID;references:ID"`
}

// ProductDescription represents a multilingual product description entity with rich metadata
type ProductDescription struct {
	ProductID   uint   `json:"ProductID" gorm:"primaryKey" odata:"key"`
	LanguageKey string `json:"LanguageKey" gorm:"primaryKey;size:2" odata:"key,maxlength=2"`
	Description string `json:"Description" gorm:"not null" odata:"required,maxlength=500"`
	LongText    string `json:"LongText" gorm:"type:text" odata:"maxlength=2000,nullable"`
	// Navigation property back to Product
	Product *Product `json:"Product,omitempty" gorm:"foreignKey:ProductID;references:ID"`
}

// GetSampleProducts returns sample product data for seeding the database
func GetSampleProducts() []Product {
	return []Product{
		{
			ID:       1,
			Name:     "Laptop",
			Price:    999.99,
			Category: "Electronics",
			Version:  1,
		},
		{
			ID:       2,
			Name:     "Wireless Mouse",
			Price:    29.99,
			Category: "Electronics",
			Version:  1,
		},
		{
			ID:       3,
			Name:     "Coffee Mug",
			Price:    15.50,
			Category: "Kitchen",
			Version:  1,
		},
		{
			ID:       4,
			Name:     "Office Chair",
			Price:    249.99,
			Category: "Furniture",
			Version:  1,
		},
		{
			ID:       5,
			Name:     "Smartphone",
			Price:    799.99,
			Category: "Electronics",
			Version:  1,
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
			LongText:    "This state-of-the-art laptop features the latest processor technology, dedicated graphics card, and ample RAM to handle all your computing needs. Perfect for both professional work and entertainment.",
		},
		{
			ProductID:   1,
			LanguageKey: "DE",
			Description: "Hochleistungs-Laptop für Produktivität und Gaming",
			LongText:    "Dieser hochmoderne Laptop verfügt über die neueste Prozessortechnologie, eine dedizierte Grafikkarte und reichlich RAM, um alle Ihre Computeranforderungen zu erfüllen. Perfekt für berufliche Arbeit und Unterhaltung.",
		},
		{
			ProductID:   2,
			LanguageKey: "EN",
			Description: "Ergonomic wireless mouse with precision tracking",
			LongText:    "Experience comfort and precision with this wireless mouse. Features adjustable DPI settings, long battery life, and a contoured design that fits perfectly in your hand.",
		},
		{
			ProductID:   2,
			LanguageKey: "FR",
			Description: "Souris sans fil ergonomique avec suivi de précision",
			LongText:    "Découvrez le confort et la précision avec cette souris sans fil. Dispose de paramètres DPI réglables, d'une longue durée de vie de la batterie et d'un design profilé qui s'adapte parfaitement à votre main.",
		},
		{
			ProductID:   3,
			LanguageKey: "EN",
			Description: "Ceramic coffee mug with heat retention technology",
			LongText:    "Keep your beverages at the perfect temperature with this innovative ceramic mug. The double-wall construction provides excellent insulation while remaining comfortable to hold.",
		},
		{
			ProductID:   5,
			LanguageKey: "EN",
			Description: "Latest generation smartphone with advanced camera",
			LongText:    "Capture life's moments in stunning detail with our flagship smartphone. Features a professional-grade camera system, lightning-fast processor, and all-day battery life.",
		},
		{
			ProductID:   5,
			LanguageKey: "ES",
			Description: "Smartphone de última generación con cámara avanzada",
			LongText:    "Captura los momentos de la vida con un detalle asombroso con nuestro smartphone insignia. Cuenta con un sistema de cámara de nivel profesional, procesador ultrarrápido y batería para todo el día.", //nolint:misspell // Spanish text
		},
	}
}
