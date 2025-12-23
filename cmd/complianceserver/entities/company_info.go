package entities

import (
	"time"

	"github.com/google/uuid"
)

// CompanyInfo represents a singleton entity for company information
type CompanyInfo struct {
	ID          uuid.UUID `json:"ID" gorm:"type:char(36);primaryKey" odata:"key,generate=uuid"`
	Name        string    `json:"Name" gorm:"not null" odata:"required,maxlength=200"`
	CEO         string    `json:"CEO" gorm:"not null" odata:"required,maxlength=100"`
	Founded     int       `json:"Founded" gorm:"not null"`
	HeadQuarter string    `json:"HeadQuarter" gorm:"not null" odata:"maxlength=200"`
	Website     string    `json:"Website" gorm:"not null" odata:"maxlength=100"`
	Logo        []byte    `json:"Logo" odata:"nullable,contenttype=image/svg+xml"` // Binary data example (company logo)
	Version     int       `json:"Version" gorm:"default:1" odata:"etag"`
	UpdatedAt   time.Time `json:"UpdatedAt" gorm:"not null"`
}

// TableName overrides the table name used by GORM to match OData singleton name
func (CompanyInfo) TableName() string {
	return "Company"
}

// GetCompanyInfo returns the singleton company information
// Note: ID is server-generated
func GetCompanyInfo() CompanyInfo {
	// Create a simple SVG logo as binary data
	svgLogo := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100" viewBox="0 0 100 100">
  <rect width="100" height="100" fill="#4A90E2"/>
  <text x="50" y="55" font-family="Arial" font-size="24" fill="white" text-anchor="middle">TS</text>
</svg>`)

	return CompanyInfo{
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
