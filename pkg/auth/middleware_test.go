package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIKeyMiddleware_Disabled(t *testing.T) {
	cfg := Config{
		Enabled: false,
	}

	middleware := APIKeyMiddleware(cfg)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 when auth disabled, got %d", w.Code)
	}
}

func TestAPIKeyMiddleware_MissingHeader(t *testing.T) {
	cfg := Config{
		Enabled: true,
		APIKeys: map[string]APIKeyInfo{},
	}

	middleware := APIKeyMiddleware(cfg)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for missing auth header, got %d", w.Code)
	}
}

func TestAPIKeyMiddleware_ValidKey(t *testing.T) {
	cfg := Config{
		Enabled: true,
		APIKeys: map[string]APIKeyInfo{
			"test-key-123": {
				Name:        "Test Key",
				Permissions: []string{"read", "write"},
				Enabled:     true,
			},
		},
	}

	middleware := APIKeyMiddleware(cfg)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

func TestAPIKeyMiddleware_InvalidKey(t *testing.T) {
	cfg := Config{
		Enabled: true,
		APIKeys: map[string]APIKeyInfo{
			"valid-key": {
				Name:    "Valid Key",
				Enabled: true,
			},
		},
	}

	middleware := APIKeyMiddleware(cfg)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

func TestAPIKeyMiddleware_DisabledKey(t *testing.T) {
	cfg := Config{
		Enabled: true,
		APIKeys: map[string]APIKeyInfo{
			"disabled-key": {
				Name:    "Disabled Key",
				Enabled: false,
			},
		},
	}

	middleware := APIKeyMiddleware(cfg)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

func TestAPIKeyMiddleware_BearerFormats(t *testing.T) {
	cfg := Config{
		Enabled: true,
		APIKeys: map[string]APIKeyInfo{
			"test-key": {
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
		{"Bearer uppercase", "Bearer test-key", http.StatusOK},
		{"bearer lowercase", "bearer test-key", http.StatusOK},
		{"plain key", "test-key", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := APIKeyMiddleware(cfg)
			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

func TestSecureCompareAPIKey(t *testing.T) {
	tests := []struct {
		key1     string
		key2     string
		expected bool
	}{
		{"test-key", "test-key", true},
		{"test-key", "different-key", false},
		{"", "", true},
		{"key", "", false},
	}

	for _, tt := range tests {
		result := SecureCompareAPIKey(tt.key1, tt.key2)
		if result != tt.expected {
			t.Errorf("SecureCompareAPIKey(%q, %q) = %v, want %v",
				tt.key1, tt.key2, result, tt.expected)
		}
	}
}

func TestValidateAPIKey(t *testing.T) {
	cfg := Config{
		Enabled: true,
		APIKeys: map[string]APIKeyInfo{
			"valid-key": {
				Name:    "Valid Key",
				Enabled: true,
			},
			"disabled-key": {
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
		{"valid key", "valid-key", true},
		{"disabled key", "disabled-key", false},
		{"invalid key", "nonexistent-key", false},
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

func TestValidateAPIKey_Disabled(t *testing.T) {
	cfg := Config{
		Enabled: false,
	}

	_, valid := ValidateAPIKey(cfg, "any-key")
	if !valid {
		t.Error("Expected validation to pass when auth is disabled")
	}
}

func TestHasPermission(t *testing.T) {
	tests := []struct {
		name       string
		keyInfo    APIKeyInfo
		permission string
		expected   bool
	}{
		{
			name: "has specific permission",
			keyInfo: APIKeyInfo{
				Permissions: []string{"read", "write"},
			},
			permission: "read",
			expected:   true,
		},
		{
			name: "no permission",
			keyInfo: APIKeyInfo{
				Permissions: []string{"read"},
			},
			permission: "write",
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
			permission: "read",
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
