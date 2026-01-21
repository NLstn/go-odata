package entities

import "github.com/google/uuid"

// ReadOnlyItem represents a read-only entity for capability restriction tests.
type ReadOnlyItem struct {
	ID   uuid.UUID `json:"ID" gorm:"type:char(36);primaryKey" odata:"key"`
	Name string    `json:"Name" gorm:"not null" odata:"required,maxlength=100"`
}

// TableName overrides the table name used by GORM to match OData entity set name.
func (ReadOnlyItem) TableName() string {
	return "ReadOnlyItems"
}

// GetSampleReadOnlyItems returns sample data for read-only items.
func GetSampleReadOnlyItems() []ReadOnlyItem {
	return []ReadOnlyItem{
		{Name: "Read-only item A"},
		{Name: "Read-only item B"},
	}
}
