package entities

import (
	"fmt"
	"time"
)

// CachedFeatureFlag represents a feature flag entity with OData full-dataset caching enabled.
type CachedFeatureFlag struct {
	ID                uint      `json:"ID" gorm:"primaryKey" odata:"key"`
	Key               string    `json:"Key" gorm:"not null;uniqueIndex:idx_cached_feature_flags_key" odata:"required,maxlength=200"`
	Description       string    `json:"Description" odata:"maxlength=500"`
	Environment       string    `json:"Environment" gorm:"not null;index:idx_cached_feature_flags_env_enabled_rollout,priority:1" odata:"required,maxlength=20"`
	Enabled           bool      `json:"Enabled" gorm:"not null;index:idx_cached_feature_flags_env_enabled_rollout,priority:2"`
	RolloutPercentage int       `json:"RolloutPercentage" gorm:"not null;index:idx_cached_feature_flags_env_enabled_rollout,priority:3,sort:desc" odata:"required"`
	Owner             string    `json:"Owner" odata:"maxlength=120"`
	UpdatedAt         time.Time `json:"UpdatedAt"`
}

// UncachedFeatureFlag represents a feature flag entity that always queries the primary database.
type UncachedFeatureFlag struct {
	ID                uint      `json:"ID" gorm:"primaryKey" odata:"key"`
	Key               string    `json:"Key" gorm:"not null;uniqueIndex:idx_uncached_feature_flags_key" odata:"required,maxlength=200"`
	Description       string    `json:"Description" odata:"maxlength=500"`
	Environment       string    `json:"Environment" gorm:"not null" odata:"required,maxlength=20"`
	Enabled           bool      `json:"Enabled" gorm:"not null"`
	RolloutPercentage int       `json:"RolloutPercentage" gorm:"not null" odata:"required"`
	Owner             string    `json:"Owner" odata:"maxlength=120"`
	UpdatedAt         time.Time `json:"UpdatedAt"`
}

func buildFeatureFlagSeed(count int) []featureFlagSeed {
	if count <= 0 {
		return nil
	}

	environments := []string{"prod", "staging", "dev", "qa"}
	owners := []string{"platform", "checkout", "search", "recommendations", "mobile", "security"}
	seed := make([]featureFlagSeed, 0, count)

	for i := 0; i < count; i++ {
		env := environments[i%len(environments)]
		rollout := (i * 7) % 101
		enabled := rollout >= 50
		seed = append(seed, featureFlagSeed{
			ID:                uint(i + 1),
			Key:               fmt.Sprintf("feature_%06d", i+1),
			Description:       fmt.Sprintf("Feature flag %d used for cache performance benchmarks", i+1),
			Environment:       env,
			Enabled:           enabled,
			RolloutPercentage: rollout,
			Owner:             owners[i%len(owners)],
			UpdatedAt:         time.Now().Add(-time.Duration(i%720) * time.Hour),
		})
	}

	return seed
}

type featureFlagSeed struct {
	ID                uint
	Key               string
	Description       string
	Environment       string
	Enabled           bool
	RolloutPercentage int
	Owner             string
	UpdatedAt         time.Time
}

// GenerateCachedFeatureFlags returns a large deterministic dataset for cached feature-flag benchmarks.
func GenerateCachedFeatureFlags(count int) []CachedFeatureFlag {
	seed := buildFeatureFlagSeed(count)
	flags := make([]CachedFeatureFlag, len(seed))
	for i := range seed {
		flags[i] = CachedFeatureFlag{
			ID:                seed[i].ID,
			Key:               seed[i].Key,
			Description:       seed[i].Description,
			Environment:       seed[i].Environment,
			Enabled:           seed[i].Enabled,
			RolloutPercentage: seed[i].RolloutPercentage,
			Owner:             seed[i].Owner,
			UpdatedAt:         seed[i].UpdatedAt,
		}
	}
	return flags
}

// GenerateUncachedFeatureFlags returns the same deterministic dataset for uncached benchmarks.
func GenerateUncachedFeatureFlags(count int) []UncachedFeatureFlag {
	seed := buildFeatureFlagSeed(count)
	flags := make([]UncachedFeatureFlag, len(seed))
	for i := range seed {
		flags[i] = UncachedFeatureFlag{
			ID:                seed[i].ID,
			Key:               seed[i].Key,
			Description:       seed[i].Description,
			Environment:       seed[i].Environment,
			Enabled:           seed[i].Enabled,
			RolloutPercentage: seed[i].RolloutPercentage,
			Owner:             seed[i].Owner,
			UpdatedAt:         seed[i].UpdatedAt,
		}
	}
	return flags
}

// GetSampleCachedFeatureFlags returns a small deterministic sample dataset.
func GetSampleCachedFeatureFlags() []CachedFeatureFlag {
	return GenerateCachedFeatureFlags(100)
}

// GetSampleUncachedFeatureFlags returns a small deterministic sample dataset.
func GetSampleUncachedFeatureFlags() []UncachedFeatureFlag {
	return GenerateUncachedFeatureFlags(100)
}
