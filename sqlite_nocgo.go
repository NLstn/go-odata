//go:build !cgo

package odata

import "gorm.io/gorm"

func ensureSQLiteRegexp(_ *gorm.DB) error {
	return nil
}