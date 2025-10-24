package entities

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

// stringPtr is a helper function to create a pointer to a string
func stringPtr(s string) *string {
	return &s
}
