package main

import (
	"context"
	"net/http"
	"strings"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// userIDContextKey is the key used to store the user ID in the request context
	userIDContextKey contextKey = "userID"
)

// authMiddleware is a dummy authentication middleware for demonstration purposes only.
// WARNING: This is NOT secure and should NOT be used in production!
//
// For demonstration purposes, this middleware:
// - Extracts the Authorization header
// - Strips the "Bearer " prefix
// - Stores the remaining value as the user ID in the request context
//
// Example usage:
//
//	Authorization: Bearer 1
//
// This would extract "1" as the user ID and store it in the context.
func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract the Authorization header
		authHeader := r.Header.Get("Authorization")

		// If the header exists and starts with "Bearer ", extract the token/user ID
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			// Strip the "Bearer " prefix (7 characters)
			userID := strings.TrimPrefix(authHeader, "Bearer ")
			userID = strings.TrimSpace(userID)

			// Store the user ID in the request context
			if userID != "" {
				ctx := context.WithValue(r.Context(), userIDContextKey, userID)
				r = r.WithContext(ctx)
			}
		}

		// Call the next handler
		next.ServeHTTP(w, r)
	})
}
