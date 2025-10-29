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
	if !pref.RespondAsyncRequested {
		t.Error("RespondAsyncRequested should be true when respond-async is present")
	}
}

func TestParsePrefer_RespondAsyncCaseInsensitive(t *testing.T) {
	headers := []string{"RESPOND-ASYNC", "Respond-Async", "ReSpOnD-AsYnC"}
	for _, header := range headers {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Prefer", header)

		pref := ParsePrefer(req)
		if !pref.RespondAsyncRequested {
			t.Fatalf("expected RespondAsyncRequested to be true for header %s", header)
		}
		if pref.RespondAsyncApplied() {
			t.Fatalf("RespondAsync should not be applied automatically for header %s", header)
		}
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
	if applied != "return=representation, return=minimal" {
		t.Errorf("Expected 'return=representation, return=minimal', got '%s'", applied)
	}
}

func TestParsePrefer_MaxPageSize(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Prefer", "odata.maxpagesize=50")
	pref := ParsePrefer(req)

	if pref.MaxPageSize == nil {
		t.Error("MaxPageSize should be set")
	} else if *pref.MaxPageSize != 50 {
		t.Errorf("Expected MaxPageSize to be 50, got %d", *pref.MaxPageSize)
	}
}

func TestParsePrefer_MaxPageSizeCaseVariations(t *testing.T) {
	testCases := []struct {
		name   string
		header string
		expect int
	}{
		{"Lowercase", "odata.maxpagesize=100", 100},
		{"CamelCase", "odata.maxPageSize=200", 200},
		{"PascalCase", "odata.MaxPageSize=300", 300},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Prefer", tc.header)
			pref := ParsePrefer(req)

			if pref.MaxPageSize == nil {
				t.Errorf("MaxPageSize should be set for %s", tc.header)
			} else if *pref.MaxPageSize != tc.expect {
				t.Errorf("Expected MaxPageSize to be %d, got %d", tc.expect, *pref.MaxPageSize)
			}
		})
	}
}

func TestParsePrefer_MaxPageSizeInvalid(t *testing.T) {
	testCases := []string{
		"odata.maxpagesize=abc",
		"odata.maxpagesize=-10",
		"odata.maxpagesize=0",
	}

	for _, tc := range testCases {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Prefer", tc)
		pref := ParsePrefer(req)

		if pref.MaxPageSize != nil {
			t.Errorf("MaxPageSize should not be set for invalid value: %s", tc)
		}
	}
}

func TestParsePrefer_TrackChanges(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Prefer", "odata.track-changes")

	pref := ParsePrefer(req)
	if !pref.TrackChangesRequested {
		t.Fatalf("expected track changes to be requested")
	}

	pref.ApplyTrackChanges()
	applied := pref.GetPreferenceApplied()
	if applied != "odata.track-changes" {
		t.Fatalf("expected Preference-Applied to include track changes, got %s", applied)
	}
}

func TestGetPreferenceApplied_TrackChangesNotApplied(t *testing.T) {
	pref := &Preference{TrackChangesRequested: true}

	if applied := pref.GetPreferenceApplied(); applied != "" {
		t.Fatalf("expected empty applied preferences, got %s", applied)
	}
}

func TestParsePrefer_MultipleWithMaxPageSize(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Prefer", "return=representation, odata.maxpagesize=25")
	pref := ParsePrefer(req)

	if !pref.ReturnRepresentation {
		t.Error("ReturnRepresentation should be true")
	}
	if pref.MaxPageSize == nil {
		t.Error("MaxPageSize should be set")
	} else if *pref.MaxPageSize != 25 {
		t.Errorf("Expected MaxPageSize to be 25, got %d", *pref.MaxPageSize)
	}
}

func TestGetPreferenceApplied_MaxPageSize(t *testing.T) {
	maxPageSize := 50
	pref := &Preference{MaxPageSize: &maxPageSize}

	applied := pref.GetPreferenceApplied()
	if applied != "odata.maxpagesize=50" {
		t.Errorf("Expected 'odata.maxpagesize=50', got '%s'", applied)
	}
}

func TestGetPreferenceApplied_MultiplePreferences(t *testing.T) {
	maxPageSize := 100
	pref := &Preference{
		ReturnRepresentation: true,
		MaxPageSize:          &maxPageSize,
	}

	applied := pref.GetPreferenceApplied()
	if applied != "return=representation, odata.maxpagesize=100" {
		t.Errorf("Expected 'return=representation, odata.maxpagesize=100', got '%s'", applied)
	}
}

func TestGetPreferenceApplied_RespondAsyncRequestedButNotApplied(t *testing.T) {
	pref := &Preference{RespondAsyncRequested: true}

	if pref.GetPreferenceApplied() != "" {
		t.Fatalf("expected no applied preferences when async is not applied")
	}
}

func TestGetPreferenceApplied_RespondAsyncApplied(t *testing.T) {
	pref := &Preference{RespondAsyncRequested: true}
	pref.ApplyRespondAsync()

	if !pref.RespondAsyncApplied() {
		t.Fatalf("expected respond-async to be marked as applied")
	}

	if applied := pref.GetPreferenceApplied(); applied != "respond-async" {
		t.Fatalf("expected Preference-Applied to be respond-async, got %s", applied)
	}
}

func TestGetPreferenceApplied_RespondAsyncWithOtherPreferences(t *testing.T) {
	maxPageSize := 10
	pref := &Preference{
		ReturnRepresentation:  true,
		MaxPageSize:           &maxPageSize,
		RespondAsyncRequested: true,
	}
	pref.ApplyRespondAsync()

	applied := pref.GetPreferenceApplied()
	expected := "return=representation, odata.maxpagesize=10, respond-async"
	if applied != expected {
		t.Fatalf("expected '%s', got '%s'", expected, applied)
	}
}

func TestApplyRespondAsyncWithoutRequest(t *testing.T) {
	pref := &Preference{}
	pref.ApplyRespondAsync()

	if pref.RespondAsyncApplied() {
		t.Fatalf("respond-async should not be applied when not requested")
	}
}

func TestSanitizeForAsyncDispatch(t *testing.T) {
	sanitized := SanitizeForAsyncDispatch("return=minimal, respond-async, odata.maxpagesize=10")
	expected := "return=minimal, odata.maxpagesize=10"
	if sanitized != expected {
		t.Fatalf("expected sanitized header '%s', got '%s'", expected, sanitized)
	}
}

func TestSanitizeForAsyncDispatch_OnlyRespondAsync(t *testing.T) {
	sanitized := SanitizeForAsyncDispatch("respond-async")
	if sanitized != "" {
		t.Fatalf("expected empty sanitized header, got '%s'", sanitized)
	}
}

func TestSanitizeForAsyncDispatch_MultipleRespondAsyncTokens(t *testing.T) {
	sanitized := SanitizeForAsyncDispatch("respond-async, return=minimal, RESPOND-ASYNC")
	if sanitized != "return=minimal" {
		t.Fatalf("expected sanitized header 'return=minimal', got '%s'", sanitized)
	}
}
