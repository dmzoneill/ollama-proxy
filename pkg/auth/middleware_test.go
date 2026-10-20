package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestAPIKeyMiddleware_Disabled is part of the public API.
func TestAPIKeyMiddleware_Disabled(t *testing.T) {
	cfg := Config{
		Enabled: false,
	}

	middleware := APIKeyMiddleware(cfg)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 when auth disabled, got %d", w.Code)
	}
}

// TestAPIKeyMiddleware_MissingHeader is part of the public API.
func TestAPIKeyMiddleware_MissingHeader(t *testing.T) {
	cfg := Config{
		Enabled: true,
		APIKeys: map[string]APIKeyInfo{},
	}

	middleware := APIKeyMiddleware(cfg)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for missing auth header, got %d", w.Code)
	}
}

// TestAPIKeyMiddleware_ValidKey is part of the public API.
func TestAPIKeyMiddleware_ValidKey(t *testing.T) {
	cfg := Config{
		Enabled: true,
		APIKeys: map[string]APIKeyInfo{
			"test-key-123": {
				Name:        "Test Key",
				Permissions: []string{strRead, strWrite},
				Enabled:     true,
			},
		},
	}

	middleware := APIKeyMiddleware(cfg)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer test-key-123")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for valid key, got %d", w.Code)
	}
}

// TestAPIKeyMiddleware_InvalidKey is part of the public API.
func TestAPIKeyMiddleware_InvalidKey(t *testing.T) {
	cfg := Config{
		Enabled: true,
		APIKeys: map[string]APIKeyInfo{
			strValidKey: {
				Name:    "Valid Key",
				Enabled: true,
			},
		},
	}

	middleware := APIKeyMiddleware(cfg)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-key")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for invalid key, got %d", w.Code)
	}
}

// TestAPIKeyMiddleware_DisabledKey is part of the public API.
func TestAPIKeyMiddleware_DisabledKey(t *testing.T) {
	cfg := Config{
		Enabled: true,
		APIKeys: map[string]APIKeyInfo{
			strDisabledKey: {
				Name:    "Disabled Key",
				Enabled: false,
			},
		},
	}

	middleware := APIKeyMiddleware(cfg)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer disabled-key")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403 for disabled key, got %d", w.Code)
	}
}

// TestAPIKeyMiddleware_BearerFormats is part of the public API.
func TestAPIKeyMiddleware_BearerFormats(t *testing.T) {
	cfg := Config{
		Enabled: true,
		APIKeys: map[string]APIKeyInfo{
			strTestKey: {
				Name:    "Test Key",
				Enabled: true,
			},
		},
	}

	tests := []struct {
		name   string
		header string
		status int
	}{
		{name: "Bearer uppercase", header: "Bearer test-key", status: http.StatusOK},
		{name: "bearer lowercase", header: "bearer test-key", status: http.StatusOK},
		{name: "plain key", header: strTestKey, status: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := APIKeyMiddleware(cfg)
			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", tt.header)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.status {
				t.Errorf("Expected status %d, got %d", tt.status, w.Code)
			}
		})
	}
}

// TestSecureCompareAPIKey is part of the public API.
func TestSecureCompareAPIKey(t *testing.T) {
	tests := []struct {
		key1     string
		key2     string
		expected bool
	}{
		{key1: strTestKey, key2: strTestKey, expected: true},
		{key1: strTestKey, key2: "different-key", expected: false},
		{key1: "", key2: "", expected: true},
		{key1: "key", key2: "", expected: false},
	}

	for _, tt := range tests {
		result := SecureCompareAPIKey(tt.key1, tt.key2)
		if result != tt.expected {
			t.Errorf("SecureCompareAPIKey(%q, %q) = %v, want %v",
				tt.key1, tt.key2, result, tt.expected)
		}
	}
}

// TestValidateAPIKey is part of the public API.
func TestValidateAPIKey(t *testing.T) {
	cfg := Config{
		Enabled: true,
		APIKeys: map[string]APIKeyInfo{
			strValidKey: {
				Name:    "Valid Key",
				Enabled: true,
			},
			strDisabledKey: {
				Name:    "Disabled Key",
				Enabled: false,
			},
		},
	}

	tests := []struct {
		name      string
		key       string
		wantValid bool
	}{
		{name: "valid key", key: strValidKey, wantValid: true},
		{name: "disabled key", key: strDisabledKey, wantValid: false},
		{name: "invalid key", key: "nonexistent-key", wantValid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, valid := ValidateAPIKey(cfg, tt.key)
			if valid != tt.wantValid {
				t.Errorf("ValidateAPIKey(%q) = %v, want %v", tt.key, valid, tt.wantValid)
			}
		})
	}
}

// TestValidateAPIKey_Disabled is part of the public API.
func TestValidateAPIKey_Disabled(t *testing.T) {
	cfg := Config{
		Enabled: false,
	}

	_, valid := ValidateAPIKey(cfg, "any-key")
	if !valid {
		t.Error("Expected validation to pass when auth is disabled")
	}
}

// TestHasPermission is part of the public API.
func TestHasPermission(t *testing.T) {
	tests := []struct {
		name       string
		permission string
		keyInfo    APIKeyInfo
		expected   bool
	}{
		{
			name: "has specific permission",
			keyInfo: APIKeyInfo{
				Permissions: []string{strRead, strWrite},
			},
			permission: strRead,
			expected:   true,
		},
		{
			name: "no permission",
			keyInfo: APIKeyInfo{
				Permissions: []string{strRead},
			},
			permission: strWrite,
			expected:   false,
		},
		{
			name: "wildcard permission",
			keyInfo: APIKeyInfo{
				Permissions: []string{"*"},
			},
			permission: "anything",
			expected:   true,
		},
		{
			name: "no permissions",
			keyInfo: APIKeyInfo{
				Permissions: []string{},
			},
			permission: strRead,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasPermission(tt.keyInfo, tt.permission)
			if result != tt.expected {
				t.Errorf("HasPermission() = %v, want %v", result, tt.expected)
			}
		})
	}
}
