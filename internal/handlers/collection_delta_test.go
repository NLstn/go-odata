package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nlstn/go-odata/internal/etag"
	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/trackchanges"
)

func TestBuildDeltaEntriesMetadataLevels(t *testing.T) {
	entityMetadata := &metadata.EntityMetadata{
		EntityName:    "Widget",
		EntitySetName: "Widgets",
		KeyProperties: []metadata.PropertyMetadata{
			{Name: "ID", JsonName: "ID", FieldName: "ID", IsKey: true},
			{Name: "Region", JsonName: "Region", FieldName: "Region", IsKey: true},
		},
		ETagProperty: &metadata.PropertyMetadata{
			Name:      "Version",
			JsonName:  "Version",
			FieldName: "Version",
			IsETag:    true,
		},
	}

	handler := NewEntityHandler(nil, entityMetadata, nil)

	events := []trackchanges.ChangeEvent{
		{
			Type: trackchanges.ChangeTypeAdded,
			KeyValues: map[string]interface{}{
				"ID":     1,
				"Region": "NA",
			},
			Data: map[string]interface{}{
				"ID":      1,
				"Region":  "NA",
				"Name":    "Alpha",
				"Version": 3,
			},
		},
		{
			Type: trackchanges.ChangeTypeUpdated,
			KeyValues: map[string]interface{}{
				"ID":     2,
				"Region": "EU",
			},
			Data: map[string]interface{}{
				"ID":      2,
				"Region":  "EU",
				"Name":    "Beta",
				"Version": 4,
			},
		},
		{
			Type: trackchanges.ChangeTypeDeleted,
			KeyValues: map[string]interface{}{
				"ID":     3,
				"Region": "APAC",
			},
		},
	}

	testCases := []struct {
		name              string
		metadataLevel     string
		includeMetadata   bool
		includeType       bool
		includeETag       bool
		addedResourceID   string
		updatedResourceID string
		deletedResourceID string
	}{
		{
			name:              "metadata none",
			metadataLevel:     "none",
			includeMetadata:   false,
			includeType:       false,
			includeETag:       false,
			addedResourceID:   "http://example.com/Widgets(ID=1,Region='NA')",
			updatedResourceID: "http://example.com/Widgets(ID=2,Region='EU')",
			deletedResourceID: "http://example.com/Widgets(ID=3,Region='APAC')",
		},
		{
			name:              "metadata minimal",
			metadataLevel:     "minimal",
			includeMetadata:   true,
			includeType:       true,
			includeETag:       true,
			addedResourceID:   "http://example.com/Widgets(ID=1,Region='NA')",
			updatedResourceID: "http://example.com/Widgets(ID=2,Region='EU')",
			deletedResourceID: "http://example.com/Widgets(ID=3,Region='APAC')",
		},
		{
			name:              "metadata full",
			metadataLevel:     "full",
			includeMetadata:   true,
			includeType:       true,
			includeETag:       true,
			addedResourceID:   "http://example.com/Widgets(ID=1,Region='NA')",
			updatedResourceID: "http://example.com/Widgets(ID=2,Region='EU')",
			deletedResourceID: "http://example.com/Widgets(ID=3,Region='APAC')",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "http://example.com/odata/Widgets", nil)
			req.Header.Set("Accept", "application/json;odata.metadata="+tc.metadataLevel)

			entries := handler.buildDeltaEntries(req, events)
			if len(entries) != len(events) {
				t.Fatalf("expected %d entries, got %d", len(events), len(entries))
			}

			expectedType := "#" + defaultNamespace + ".Widget"

			assertMetadata := func(entry map[string]interface{}, resourceID string, expectedETag string) {
				if tc.includeMetadata {
					if entry["@odata.id"] != resourceID {
						t.Fatalf("expected @odata.id %q, got %v", resourceID, entry["@odata.id"])
					}
				} else if _, ok := entry["@odata.id"]; ok {
					t.Fatalf("expected no @odata.id, got %v", entry["@odata.id"])
				}

				if tc.includeType {
					if entry["@odata.type"] != expectedType {
						t.Fatalf("expected @odata.type %s, got %v", expectedType, entry["@odata.type"])
					}
				} else if _, ok := entry["@odata.type"]; ok {
					t.Fatalf("expected no @odata.type, got %v", entry["@odata.type"])
				}

				if tc.includeETag {
					if entry["@odata.etag"] != expectedETag {
						t.Fatalf("expected @odata.etag %q, got %v", expectedETag, entry["@odata.etag"])
					}
				} else if _, ok := entry["@odata.etag"]; ok {
					t.Fatalf("expected no @odata.etag, got %v", entry["@odata.etag"])
				}
			}

			addedEntry := entries[0]
			updatedEntry := entries[1]
			deletedEntry := entries[2]

			assertMetadata(addedEntry, tc.addedResourceID, etag.Generate(events[0].Data, entityMetadata))
			assertMetadata(updatedEntry, tc.updatedResourceID, etag.Generate(events[1].Data, entityMetadata))

			if tc.includeMetadata {
				if deletedEntry["@odata.id"] != tc.deletedResourceID {
					t.Fatalf("expected deleted @odata.id %q, got %v", tc.deletedResourceID, deletedEntry["@odata.id"])
				}
			} else if _, ok := deletedEntry["@odata.id"]; ok {
				t.Fatalf("expected no deleted @odata.id, got %v", deletedEntry["@odata.id"])
			}

			if tc.includeType {
				if deletedEntry["@odata.type"] != expectedType {
					t.Fatalf("expected deleted @odata.type %s, got %v", expectedType, deletedEntry["@odata.type"])
				}
			} else if _, ok := deletedEntry["@odata.type"]; ok {
				t.Fatalf("expected no deleted @odata.type, got %v", deletedEntry["@odata.type"])
			}

			if _, ok := deletedEntry["@odata.etag"]; ok {
				t.Fatalf("expected no deleted @odata.etag, got %v", deletedEntry["@odata.etag"])
			}

			removed, ok := deletedEntry["@odata.removed"].(map[string]string)
			if !ok {
				t.Fatalf("expected @odata.removed to be map[string]string, got %T", deletedEntry["@odata.removed"])
			}
			if removed["reason"] != "deleted" {
				t.Fatalf("expected @odata.removed reason 'deleted', got %q", removed["reason"])
			}
			if deletedEntry["ID"] != 3 || deletedEntry["Region"] != "APAC" {
				t.Fatalf("expected deleted key values preserved, got ID=%v Region=%v", deletedEntry["ID"], deletedEntry["Region"])
			}
		})
	}
}
