package entities

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// DecimalSample is a dedicated compliance entity for Edm.Decimal behavior.
// It isolates decimal tests from Product.Price, which is Edm.Double.
type DecimalSample struct {
	ID     uuid.UUID       `json:"ID" gorm:"type:char(36);primaryKey" odata:"key,generate=uuid"`
	Name   string          `json:"Name" gorm:"not null" odata:"required,maxlength=100"`
	Amount decimal.Decimal `json:"Amount" gorm:"type:decimal(38,18);not null" odata:"required,precision=38,scale=18"`
}

// TableName overrides the table name used by GORM to match OData entity set name.
func (DecimalSample) TableName() string {
	return "DecimalSamples"
}

// GetSampleDecimalSamples returns seed data for decimal compliance tests.
func GetSampleDecimalSamples() []DecimalSample {
	return []DecimalSample{
		{
			Name:   "Seed Decimal",
			Amount: decimal.RequireFromString("123.450000000000000000"),
		},
	}
}
