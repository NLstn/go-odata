package odata_test

import (
	"testing"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// NamespaceTestProduct is a test entity for SetNamespace tests
type NamespaceTestProduct struct {
	ID   int    `json:"id" gorm:"primarykey" odata:"key"`
	Name string `json:"name"`
}

func setupSetNamespaceTestService(t *testing.T) (*odata.Service, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&NamespaceTestProduct{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service := odata.NewService(db)
	if err := service.RegisterEntity(NamespaceTestProduct{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	return service, db
}

func TestSetNamespace_ValidNamespace(t *testing.T) {
	service, _ := setupSetNamespaceTestService(t)

	err := service.SetNamespace("MyCompany.API")
	if err != nil {
		t.Errorf("SetNamespace failed with valid namespace: %v", err)
	}
}

func TestSetNamespace_EmptyString(t *testing.T) {
	service, _ := setupSetNamespaceTestService(t)

	err := service.SetNamespace("")
	if err == nil {
		t.Error("SetNamespace should fail with empty string")
	}
}

func TestSetNamespace_WhitespaceOnly(t *testing.T) {
	service, _ := setupSetNamespaceTestService(t)

	err := service.SetNamespace("   ")
	if err == nil {
		t.Error("SetNamespace should fail with whitespace-only string")
	}
}

func TestSetNamespace_TrimsWhitespace(t *testing.T) {
	service, _ := setupSetNamespaceTestService(t)

	// Should succeed with namespace that has leading/trailing whitespace
	err := service.SetNamespace("  ValidNamespace  ")
	if err != nil {
		t.Errorf("SetNamespace failed with whitespace-padded namespace: %v", err)
	}
}

func TestSetNamespace_SameNamespaceTwice(t *testing.T) {
	service, _ := setupSetNamespaceTestService(t)

	// First call should succeed
	err := service.SetNamespace("TestNamespace")
	if err != nil {
		t.Errorf("First SetNamespace call failed: %v", err)
	}

	// Second call with same namespace should succeed (no-op)
	err = service.SetNamespace("TestNamespace")
	if err != nil {
		t.Errorf("Second SetNamespace call with same namespace failed: %v", err)
	}
}

func TestSetNamespace_DifferentNamespaces(t *testing.T) {
	service, _ := setupSetNamespaceTestService(t)

	// First namespace
	err := service.SetNamespace("FirstNamespace")
	if err != nil {
		t.Errorf("First SetNamespace call failed: %v", err)
	}

	// Change to different namespace
	err = service.SetNamespace("SecondNamespace")
	if err != nil {
		t.Errorf("Second SetNamespace call with different namespace failed: %v", err)
	}
}

func TestSetNamespace_BeforeEntityRegistration(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&NamespaceTestProduct{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service := odata.NewService(db)

	// Set namespace before registering entities
	err = service.SetNamespace("EarlyNamespace")
	if err != nil {
		t.Errorf("SetNamespace before entity registration failed: %v", err)
	}

	// Now register entity
	if err := service.RegisterEntity(NamespaceTestProduct{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}
}

func TestSetNamespace_VariousFormats(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		wantErr   bool
	}{
		{"Simple", "MyNamespace", false},
		{"With dots", "My.Namespace.API", false},
		{"With underscore", "My_Namespace", false},
		{"Single char", "X", false},
		{"Numbers", "Namespace123", false},
		{"Empty", "", true},
		{"Whitespace", "   ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
			if err != nil {
				t.Fatalf("Failed to connect to database: %v", err)
			}
			service := odata.NewService(db)

			err = service.SetNamespace(tt.namespace)
			if tt.wantErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
