package response

import (
	"sync"
	"testing"
)

func TestSetBasePath_Concurrent(t *testing.T) {
	// Reset state
	SetBasePath("")

	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)

	// Concurrent writes
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			SetBasePath("/odata")
		}()
	}

	wg.Wait()

	// Verify final state
	if got := getBasePath(); got != "/odata" {
		t.Errorf("getBasePath() = %q, want /odata", got)
	}
}

func TestGetBasePath_Concurrent(t *testing.T) {
	SetBasePath("/api/odata")

	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)

	// Concurrent reads
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			if got := getBasePath(); got != "/api/odata" {
				t.Errorf("getBasePath() = %q, want /api/odata", got)
			}
		}()
	}

	wg.Wait()
}

func TestBasePath_EmptyString(t *testing.T) {
	SetBasePath("/odata")
	SetBasePath("") // Reset to empty

	if got := getBasePath(); got != "" {
		t.Errorf("getBasePath() = %q, want empty string", got)
	}
}
