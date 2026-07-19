package handlers

import (
	"errors"
	"fmt"
	"testing"

	"gorm.io/gorm"
)

func TestIsForeignKeyConstraintViolation(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "gorm sentinel", err: gorm.ErrForeignKeyViolated, want: true},
		{name: "wrapped gorm sentinel", err: fmt.Errorf("delete failed: %w", gorm.ErrForeignKeyViolated), want: true},
		{
			name: "postgres",
			err:  errors.New(`ERROR: update or delete on table "Products" violates foreign key constraint "fk_Products_descriptions" on table "ProductDescriptions" (SQLSTATE 23503)`),
			want: true,
		},
		{
			name: "mysql",
			err:  errors.New("Error 1451 (23000): Cannot delete or update a parent row: a foreign key constraint fails"),
			want: true,
		},
		{
			name: "sqlite",
			err:  errors.New("FOREIGN KEY constraint failed"),
			want: true,
		},
		{
			name: "sqlserver",
			err:  errors.New("The DELETE statement conflicted with the REFERENCE constraint \"FK_Products\". The conflict occurred in database ..."),
			want: true,
		},
		{
			name: "unique violation is not a foreign key violation",
			err:  errors.New("UNIQUE constraint failed: Products.id"),
			want: false,
		},
		{name: "unrelated error", err: errors.New("connection refused"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isForeignKeyConstraintViolation(tt.err); got != tt.want {
				t.Errorf("isForeignKeyConstraintViolation(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
