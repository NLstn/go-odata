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

// ============================================================================
// ⚠️  CRITICAL SECURITY WARNING ⚠️
// ============================================================================
//
// THIS AUTHENTICATION MIDDLEWARE IS COMPLETELY INSECURE AND IS FOR
// DEVELOPMENT/DEMONSTRATION PURPOSES ONLY!
//
// **DO NOT USE THIS CODE IN PRODUCTION UNDER ANY CIRCUMSTANCES!**
//
// This middleware has the following critical security vulnerabilities:
//
// 1. NO VERIFICATION: Accepts any value as a valid user ID without verification
// 2. AUTHENTICATION BYPASS: Anyone can impersonate any user
// 3. NO AUTHORIZATION: Does not validate user permissions or roles
// 4. NO ENCRYPTION: Does not validate tokens or signatures
//
// For production systems, implement proper authentication such as:
// - JWT with cryptographic signature verification
// - OAuth2 with token validation
// - Session-based authentication with secure session storage
// - API keys with proper hashing and validation
//
// See SECURITY.md for secure implementation examples.
// ============================================================================

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
//
// SECURITY RISK: This allows anyone to impersonate any user by simply
// providing any value in the Authorization header!
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
