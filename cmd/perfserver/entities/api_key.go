package entities

import (
	"fmt"
	"time"
)

// APIKey models an API credential with UUID keys generated on the server.
type APIKey struct {
	KeyID       string     `json:"KeyID" gorm:"type:char(36);primaryKey" odata:"key,generate=uuid"`
	Owner       string     `json:"Owner" gorm:"not null" odata:"required,maxlength=100"`
	Description string     `json:"Description" odata:"maxlength=200"`
	CreatedAt   time.Time  `json:"CreatedAt" gorm:"not null"`
	LastUsedAt  *time.Time `json:"LastUsedAt" odata:"nullable"`
}

// GetSampleAPIKeys returns a small dataset for development/performance warmups.
func GetSampleAPIKeys() []APIKey {
	now := time.Now().UTC()
	lastWeek := now.Add(-7 * 24 * time.Hour)

	return []APIKey{
		{
			KeyID:       "d7a986a3-c5e8-4c43-81b3-3e9ca8e52e01",
			Owner:       "Benchmark Runner",
			Description: "Synthetic traffic generator",
			CreatedAt:   now.Add(-120 * 24 * time.Hour),
			LastUsedAt:  &now,
		},
		{
			KeyID:       "98d5d3fa-2c0c-4b6f-8074-936964d04e9a",
			Owner:       "Load Tester",
			Description: "Long running soak tests",
			CreatedAt:   now.Add(-45 * 24 * time.Hour),
			LastUsedAt:  &lastWeek,
		},
	}
}

// GenerateExtensiveAPIKeys produces many API keys for large dataset benchmarks.
func GenerateExtensiveAPIKeys(count int) []APIKey {
	keys := make([]APIKey, 0, count)
	now := time.Now().UTC()
	for i := 0; i < count; i++ {
		createdAt := now.Add(-time.Duration(i%365) * 24 * time.Hour)
		seq := i + 1
		keys = append(keys, APIKey{
			KeyID:       fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", seq, seq&0xffff, (seq>>4)&0xffff, (seq>>8)&0xffff, seq),
			Owner:       fmt.Sprintf("Client %04d", i+1),
			Description: "Generated for throughput profiling",
			CreatedAt:   createdAt,
		})
	}
	return keys
}
