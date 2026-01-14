package auth

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// Config holds authentication configuration
type Config struct {
	Enabled bool
	APIKeys map[string]APIKeyInfo // key -> metadata
}

// APIKeyInfo holds metadata about an API key
type APIKeyInfo struct {
	Name        string
	Permissions []string
	Enabled     bool
}

// APIKeyMiddleware creates HTTP middleware for API key authentication
func APIKeyMiddleware(cfg Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip authentication if disabled
			if !cfg.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			// Extract API key from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
				return
			}

			// Support both "Bearer <key>" and plain key formats
			key := authHeader
			if strings.HasPrefix(authHeader, "Bearer ") {
				key = strings.TrimPrefix(authHeader, "Bearer ")
			} else if strings.HasPrefix(authHeader, "bearer ") {
				key = strings.TrimPrefix(authHeader, "bearer ")
			}

			// Validate API key
			keyInfo, valid := cfg.APIKeys[key]
			if !valid {
				http.Error(w, "Invalid API key", http.StatusUnauthorized)
				return
			}

			// Check if key is enabled
			if !keyInfo.Enabled {
				http.Error(w, "API key is disabled", http.StatusForbidden)
				return
			}

			// Key is valid, proceed
			next.ServeHTTP(w, r)
		})
	}
}

// SecureCompareAPIKey performs constant-time comparison of API keys
func SecureCompareAPIKey(key1, key2 string) bool {
	return subtle.ConstantTimeCompare([]byte(key1), []byte(key2)) == 1
}

// ValidateAPIKey checks if an API key is valid
func ValidateAPIKey(cfg Config, key string) (APIKeyInfo, bool) {
	if !cfg.Enabled {
		return APIKeyInfo{}, true
	}

	keyInfo, exists := cfg.APIKeys[key]
	if !exists {
		return APIKeyInfo{}, false
	}

	if !keyInfo.Enabled {
		return keyInfo, false
	}

	return keyInfo, true
}

// HasPermission checks if an API key has a specific permission
func HasPermission(keyInfo APIKeyInfo, permission string) bool {
	for _, p := range keyInfo.Permissions {
		if p == permission || p == "*" {
			return true
		}
	}
	return false
}
