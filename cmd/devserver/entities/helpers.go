package entities

import (
	"fmt"
	"net/http"
	"strconv"

	"gorm.io/gorm"
)

// dbGetter is a function that returns the global database instance
var dbGetter func() *gorm.DB

// SetDBGetter sets the function used to retrieve the global database instance
func SetDBGetter(getter func() *gorm.DB) {
	dbGetter = getter
}

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// userIDContextKey is the key used to store the user ID in the request context
	userIDContextKey contextKey = "userID"
)

// stringPtr is a helper function to create a pointer to a string
func stringPtr(s string) *string {
	return &s
}

// GetAuthenticatedUser extracts the user ID from the request context,
// loads the corresponding user from the database, and returns it.
//
// This function works in conjunction with the authMiddleware that stores
// the user ID in the request context. It uses the global database instance
// from the devserver package.
//
// Returns:
//   - *User: The authenticated user if found
//   - error: An error if the user ID is not in the context, invalid, or the user is not found
//
// Example usage:
//
//	user, err := entities.GetAuthenticatedUser(r)
//	if err != nil {
//	    http.Error(w, "Unauthorized", http.StatusUnauthorized)
//	    return
//	}
//	// Use the authenticated user...
func GetAuthenticatedUser(r *http.Request) (*User, error) {
	// Extract the user ID from the context
	userIDStr, ok := r.Context().Value(userIDContextKey).(string)
	if !ok || userIDStr == "" {
		return nil, fmt.Errorf("user ID not found in request context")
	}

	// Convert the user ID string to uint
	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID format: %w", err)
	}

	// Get the database instance
	if dbGetter == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	db := dbGetter()

	// Load the user from the database
	var user User
	if err := db.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found with ID: %d", userID)
		}
		return nil, fmt.Errorf("failed to load user: %w", err)
	}

	return &user, nil
}
