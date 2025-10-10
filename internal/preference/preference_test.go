package preference

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParsePrefer_NoHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	pref := ParsePrefer(req)

	if pref.ReturnRepresentation {
		t.Error("ReturnRepresentation should be false when no Prefer header is present")
	}
	if pref.ReturnMinimal {
		t.Error("ReturnMinimal should be false when no Prefer header is present")
	}
}

func TestParsePrefer_ReturnRepresentation(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Prefer", "return=representation")
	pref := ParsePrefer(req)

	if !pref.ReturnRepresentation {
		t.Error("ReturnRepresentation should be true")
	}
	if pref.ReturnMinimal {
		t.Error("ReturnMinimal should be false")
	}
}

func TestParsePrefer_ReturnMinimal(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Prefer", "return=minimal")
	pref := ParsePrefer(req)

	if pref.ReturnRepresentation {
		t.Error("ReturnRepresentation should be false")
	}
	if !pref.ReturnMinimal {
		t.Error("ReturnMinimal should be true")
	}
}

func TestParsePrefer_CaseInsensitive(t *testing.T) {
	testCases := []string{
		"return=REPRESENTATION",
		"RETURN=representation",
		"Return=Representation",
	}

	for _, tc := range testCases {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Prefer", tc)
		pref := ParsePrefer(req)

		if !pref.ReturnRepresentation {
			t.Errorf("ReturnRepresentation should be true for header: %s", tc)
		}
	}
}

func TestParsePrefer_MultiplePreferences(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Prefer", "return=representation, respond-async")
	pref := ParsePrefer(req)

	if !pref.ReturnRepresentation {
		t.Error("ReturnRepresentation should be true")
	}
}

func TestParsePrefer_WithSpaces(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Prefer", "  return=representation  ")
	pref := ParsePrefer(req)

	if !pref.ReturnRepresentation {
		t.Error("ReturnRepresentation should be true even with spaces")
	}
}

func TestShouldReturnContent_PostDefault(t *testing.T) {
	pref := &Preference{}

	if !pref.ShouldReturnContent(true) {
		t.Error("POST operations should return content by default")
	}
}

func TestShouldReturnContent_PostWithMinimal(t *testing.T) {
	pref := &Preference{ReturnMinimal: true}

	if pref.ShouldReturnContent(true) {
		t.Error("POST with return=minimal should not return content")
	}
}

func TestShouldReturnContent_PatchDefault(t *testing.T) {
	pref := &Preference{}

	if pref.ShouldReturnContent(false) {
		t.Error("PATCH/PUT operations should not return content by default")
	}
}

func TestShouldReturnContent_PatchWithRepresentation(t *testing.T) {
	pref := &Preference{ReturnRepresentation: true}

	if !pref.ShouldReturnContent(false) {
		t.Error("PATCH/PUT with return=representation should return content")
	}
}

func TestGetPreferenceApplied_Representation(t *testing.T) {
	pref := &Preference{ReturnRepresentation: true}

	applied := pref.GetPreferenceApplied()
	if applied != "return=representation" {
		t.Errorf("Expected 'return=representation', got '%s'", applied)
	}
}

func TestGetPreferenceApplied_Minimal(t *testing.T) {
	pref := &Preference{ReturnMinimal: true}

	applied := pref.GetPreferenceApplied()
	if applied != "return=minimal" {
		t.Errorf("Expected 'return=minimal', got '%s'", applied)
	}
}

func TestGetPreferenceApplied_None(t *testing.T) {
	pref := &Preference{}

	applied := pref.GetPreferenceApplied()
	if applied != "" {
		t.Errorf("Expected empty string, got '%s'", applied)
	}
}

func TestGetPreferenceApplied_PriorityRepresentation(t *testing.T) {
	// If both are set, representation takes priority
	pref := &Preference{
		ReturnRepresentation: true,
		ReturnMinimal:        true,
	}

	applied := pref.GetPreferenceApplied()
	if applied != "return=representation" {
		t.Errorf("Expected 'return=representation' to take priority, got '%s'", applied)
	}
}
