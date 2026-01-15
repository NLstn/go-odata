package metadata

import "testing"

func resetKeyGeneratorNames(t *testing.T) {
	t.Helper()

	keyGeneratorNamesMu.Lock()
	original := make(map[string]struct{}, len(keyGeneratorNames))
	for name := range keyGeneratorNames {
		original[name] = struct{}{}
	}
	keyGeneratorNames = make(map[string]struct{})
	keyGeneratorNamesMu.Unlock()

	t.Cleanup(func() {
		keyGeneratorNamesMu.Lock()
		keyGeneratorNames = make(map[string]struct{}, len(original))
		for name := range original {
			keyGeneratorNames[name] = struct{}{}
		}
		keyGeneratorNamesMu.Unlock()
	})
}

func TestRegisterKeyGeneratorNameNormalizesAndIgnoresEmpty(t *testing.T) {
	resetKeyGeneratorNames(t)

	RegisterKeyGeneratorName("  FoO  ")
	RegisterKeyGeneratorName("")
	RegisterKeyGeneratorName("   ")

	if !KnownKeyGeneratorName("foo") {
		t.Fatalf("expected normalized name to be registered")
	}

	keyGeneratorNamesMu.RLock()
	defer keyGeneratorNamesMu.RUnlock()

	if len(keyGeneratorNames) != 1 {
		t.Fatalf("expected 1 registered name, got %d", len(keyGeneratorNames))
	}
	if _, ok := keyGeneratorNames["foo"]; !ok {
		t.Fatalf("expected normalized name to be stored")
	}
}

func TestKnownKeyGeneratorNameMatchesCaseInsensitive(t *testing.T) {
	resetKeyGeneratorNames(t)

	RegisterKeyGeneratorName("CuStOm")

	if !KnownKeyGeneratorName(" custom ") {
		t.Fatalf("expected case-insensitive name match")
	}
	if KnownKeyGeneratorName("unknown") {
		t.Fatalf("expected unknown name to return false")
	}
}
