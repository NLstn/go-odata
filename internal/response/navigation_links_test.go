package response

import (
	"encoding/json"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/nlstn/go-odata/internal/query"
)

func TestAddNavigationLinksWithNilData(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/Products", nil)
	result := addNavigationLinks(nil, nil, nil, nil, req, "Products", "minimal", nil)

	if result == nil {
		t.Fatal("addNavigationLinks should not return nil for nil data")
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	if string(data) != "[]" {
		t.Fatalf("expected [], got %s", string(data))
	}
}

func TestAddNavigationLinksWithEmptySlice(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/Products", nil)
	result := addNavigationLinks([]interface{}{}, nil, nil, nil, req, "Products", "minimal", nil)

	if result == nil {
		t.Fatal("addNavigationLinks should not return nil for empty slice")
	}
	if len(result) != 0 {
		t.Fatalf("expected empty slice, got %d", len(result))
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	if string(data) != "[]" {
		t.Fatalf("expected [], got %s", string(data))
	}
}

func TestAddNavigationLinksWithNonSliceData(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/Products", nil)
	single := map[string]interface{}{"ID": 1}
	result := addNavigationLinks(single, nil, nil, nil, req, "Products", "minimal", nil)

	if result == nil {
		t.Fatal("addNavigationLinks should return empty slice for non-slice data")
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	if string(data) != "[]" {
		t.Fatalf("expected [], got %s", string(data))
	}
}

func TestTruncateExpandedCollectionToTop(t *testing.T) {
	t.Run("truncates when collection exceeds top", func(t *testing.T) {
		items := []int{1, 2, 3, 4}
		val := reflect.ValueOf(items)
		result, truncated := TruncateExpandedCollectionToTop(val, 3)
		if !truncated {
			t.Fatal("expected truncated=true")
		}
		if result.Len() != 3 {
			t.Fatalf("expected length 3 after truncation, got %d", result.Len())
		}
	})

	t.Run("does not truncate when collection equals top", func(t *testing.T) {
		items := []int{1, 2, 3}
		val := reflect.ValueOf(items)
		result, truncated := TruncateExpandedCollectionToTop(val, 3)
		if truncated {
			t.Fatal("expected truncated=false when Len == top")
		}
		if result.Len() != 3 {
			t.Fatalf("expected length 3, got %d", result.Len())
		}
	})

	t.Run("does not truncate when collection is smaller than top", func(t *testing.T) {
		items := []int{1, 2}
		val := reflect.ValueOf(items)
		result, truncated := TruncateExpandedCollectionToTop(val, 5)
		if truncated {
			t.Fatal("expected truncated=false when Len < top")
		}
		if result.Len() != 2 {
			t.Fatalf("expected length 2, got %d", result.Len())
		}
	})

	t.Run("handles nil pointer", func(t *testing.T) {
		var s *[]int
		val := reflect.ValueOf(s)
		_, truncated := TruncateExpandedCollectionToTop(val, 3)
		if truncated {
			t.Fatal("expected truncated=false for nil pointer")
		}
	})
}

func TestBuildExpandedCollectionNextLink(t *testing.T) {
	top := 5
	expandOpt := &query.ExpandOption{Top: &top}

	result := BuildExpandedCollectionNextLink("http://svc", "Products", "1", "Descriptions", expandOpt)
	expected := "http://svc/Products(1)/Descriptions?$skip=5"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestBuildExpandedCollectionNextLinkWithSkip(t *testing.T) {
	top := 5
	skip := 10
	expandOpt := &query.ExpandOption{Top: &top, Skip: &skip}

	result := BuildExpandedCollectionNextLink("http://svc", "Products", "1", "Descriptions", expandOpt)
	expected := "http://svc/Products(1)/Descriptions?$skip=15"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}
