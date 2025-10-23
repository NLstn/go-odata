package entities

import "time"

// CompanyInfo represents a singleton entity for company information
// This demonstrates the singleton feature - a single entity accessed directly by name
type CompanyInfo struct {
	ID          uint      `json:"ID" gorm:"primaryKey" odata:"key"`
	Name        string    `json:"Name" gorm:"not null" odata:"required,maxlength=200"`
	CEO         string    `json:"CEO" gorm:"not null" odata:"required,maxlength=100"`
	Founded     int       `json:"Founded" gorm:"not null"`
	HeadQuarter string    `json:"HeadQuarter" gorm:"not null" odata:"maxlength=200"`
	Website     string    `json:"Website" gorm:"not null" odata:"maxlength=100"`
	Logo        []byte    `json:"Logo" odata:"nullable,contenttype=image/svg+xml"` // Binary data example (company logo) - GORM will use appropriate type (blob for SQLite, bytea for PostgreSQL)
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
