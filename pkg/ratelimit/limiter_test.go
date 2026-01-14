package ratelimit

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func TestNewIPRateLimiter(t *testing.T) {
	rl := NewIPRateLimiter(10, 20)

	if rl == nil {
		t.Fatal("Expected rate limiter to be created")
	}

	if rl.rate != 10 {
		t.Errorf("Expected rate 10, got %v", rl.rate)
	}

	if rl.burst != 20 {
		t.Errorf("Expected burst 20, got %d", rl.burst)
	}
}

func TestIPRateLimiter_Allow(t *testing.T) {
	// Create limiter with 5 requests per second, burst of 5
	rl := NewIPRateLimiter(5, 5)

	ip := "192.168.1.1"

	// Should allow first 5 requests (burst)
	for i := 0; i < 5; i++ {
		if !rl.Allow(ip) {
			t.Errorf("Request %d should be allowed (within burst)", i+1)
		}
	}

	// 6th request should be blocked
	if rl.Allow(ip) {
		t.Error("Request 6 should be blocked (exceeded burst)")
	}

	// Wait for rate limiter to refill (200ms = 1 request at 5/sec)
	time.Sleep(250 * time.Millisecond)

	// Should allow 1 more request
	if !rl.Allow(ip) {
		t.Error("Request should be allowed after waiting for refill")
	}
}

func TestIPRateLimiter_MultipleIPs(t *testing.T) {
	rl := NewIPRateLimiter(2, 2)

	ip1 := "192.168.1.1"
	ip2 := "192.168.1.2"

	// Exhaust ip1's burst
	rl.Allow(ip1)
	rl.Allow(ip1)

	// ip1 should be blocked
	if rl.Allow(ip1) {
		t.Error("ip1 should be blocked after exhausting burst")
	}

	// ip2 should still be allowed (separate limiter)
	if !rl.Allow(ip2) {
		t.Error("ip2 should be allowed (separate limiter)")
	}
}

func TestIPRateLimiter_Middleware(t *testing.T) {
	rl := NewIPRateLimiter(2, 2)

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	// Create test server
	server := httptest.NewServer(handler)
	defer server.Close()

	// Make requests
	for i := 0; i < 2; i++ {
		resp, err := http.Get(server.URL)
		if err != nil {
			t.Fatalf("Request %d failed: %v", i+1, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Request %d: expected status 200, got %d", i+1, resp.StatusCode)
		}
		resp.Body.Close()
	}

	// 3rd request should be rate limited
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Request 3 failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("Request 3: expected status 429, got %d", resp.StatusCode)
	}
}

func TestGetIP(t *testing.T) {
	tests := []struct {
		name           string
		remoteAddr     string
		xForwardedFor  string
		xRealIP        string
		expectedIP     string
	}{
		{
			name:       "direct connection",
			remoteAddr: "192.168.1.1:12345",
			expectedIP: "192.168.1.1",
		},
		{
			name:          "X-Forwarded-For single",
			remoteAddr:    "10.0.0.1:12345",
			xForwardedFor: "203.0.113.1",
			expectedIP:    "203.0.113.1",
		},
		{
			name:          "X-Forwarded-For multiple",
			remoteAddr:    "10.0.0.1:12345",
			xForwardedFor: "203.0.113.1, 198.51.100.1",
			expectedIP:    "203.0.113.1",
		},
		{
			name:       "X-Real-IP",
			remoteAddr: "10.0.0.1:12345",
			xRealIP:    "203.0.113.1",
			expectedIP: "203.0.113.1",
		},
		{
			name:          "X-Forwarded-For takes precedence over X-Real-IP",
			remoteAddr:    "10.0.0.1:12345",
			xForwardedFor: "203.0.113.1",
			xRealIP:       "198.51.100.1",
			expectedIP:    "203.0.113.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xForwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}
			if tt.xRealIP != "" {
				req.Header.Set("X-Real-IP", tt.xRealIP)
			}

			ip := getIP(req)
			if ip != tt.expectedIP {
				t.Errorf("Expected IP %s, got %s", tt.expectedIP, ip)
			}
		})
	}
}

func TestParseForwardedFor(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected []string
	}{
		{
			name:     "single IP",
			header:   "203.0.113.1",
			expected: []string{"203.0.113.1"},
		},
		{
			name:     "multiple IPs",
			header:   "203.0.113.1, 198.51.100.1, 192.0.2.1",
			expected: []string{"203.0.113.1", "198.51.100.1", "192.0.2.1"},
		},
		{
			name:     "IPs with spaces",
			header:   "203.0.113.1 , 198.51.100.1 , 192.0.2.1",
			expected: []string{"203.0.113.1", "198.51.100.1", "192.0.2.1"},
		},
		{
			name:     "empty string",
			header:   "",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseForwardedFor(tt.header)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d IPs, got %d", len(tt.expected), len(result))
				return
			}
			for i, ip := range result {
				if ip != tt.expected[i] {
					t.Errorf("Expected IP[%d] = %s, got %s", i, tt.expected[i], ip)
				}
			}
		})
	}
}

func TestCleanup(t *testing.T) {
	rl := NewIPRateLimiter(10, 10)
	rl.expiryDuration = 100 * time.Millisecond

	// Add a limiter
	ip := "192.168.1.1"
	rl.Allow(ip)

	// Verify it exists
	rl.mu.RLock()
	if _, exists := rl.limiters[ip]; !exists {
		t.Fatal("Limiter should exist after Allow()")
	}
	rl.mu.RUnlock()

	// Wait for expiry
	time.Sleep(150 * time.Millisecond)

	// Manually trigger cleanup
	rl.cleanup()

	// Verify it was removed
	rl.mu.RLock()
	if _, exists := rl.limiters[ip]; exists {
		t.Error("Limiter should be removed after expiry")
	}
	rl.mu.RUnlock()
}

func TestTrimSpace(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"test", "test"},
		{" test", "test"},
		{"test ", "test"},
		{" test ", "test"},
		{"  test  ", "test"},
		{"\ttest\t", "test"},
		{"", ""},
		{"   ", ""},
	}

	for _, tt := range tests {
		result := trimSpace(tt.input)
		if result != tt.expected {
			t.Errorf("trimSpace(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// TestIPRateLimiter_BurstHandling tests burst capacity behavior
func TestIPRateLimiter_BurstHandling(t *testing.T) {
	rl := NewIPRateLimiter(1, 3) // 1 req/sec with burst of 3
	ip := "192.168.1.1"

	// Exhaust burst capacity
	for i := 0; i < 3; i++ {
		if !rl.Allow(ip) {
			t.Errorf("Request %d should be allowed within burst of 3", i+1)
		}
	}

	// 4th request should be blocked
	if rl.Allow(ip) {
		t.Error("4th request should be blocked (burst exhausted)")
	}

	// 5th request should also be blocked
	if rl.Allow(ip) {
		t.Error("5th request should be blocked (burst exhausted)")
	}

	// Wait for 1 token to be added
	time.Sleep(1100 * time.Millisecond)

	// Should allow 1 request
	if !rl.Allow(ip) {
		t.Error("Request should be allowed after rate limit refill")
	}

	// Immediately after, should be blocked again
	if rl.Allow(ip) {
		t.Error("Request should be blocked (refilled 1 token consumed)")
	}
}

// TestIPRateLimiter_RapidBurstRequests tests rapid consecutive burst requests
func TestIPRateLimiter_RapidBurstRequests(t *testing.T) {
	rl := NewIPRateLimiter(10, 5)
	ip := "192.168.1.100"

	// Send all burst requests rapidly
	for i := 0; i < 5; i++ {
		if !rl.Allow(ip) {
			t.Errorf("Burst request %d should be allowed", i+1)
		}
	}

	// 6th should be denied
	if rl.Allow(ip) {
		t.Error("Request 6 should be denied (burst exhausted)")
	}
}

// TestIPRateLimiter_EdgeCaseBurst tests zero and very high burst values
func TestIPRateLimiter_EdgeCaseBurst(t *testing.T) {
	// Burst of 1 means only 1 request allowed at a time
	rl := NewIPRateLimiter(1, 1)
	ip := "192.168.1.200"

	if !rl.Allow(ip) {
		t.Error("First request should be allowed with burst of 1")
	}

	if rl.Allow(ip) {
		t.Error("Second request should be denied with burst of 1")
	}
}

// TestIPRateLimiter_HighBurst tests very high burst capacity
func TestIPRateLimiter_HighBurst(t *testing.T) {
	rl := NewIPRateLimiter(1, 100)
	ip := "192.168.1.50"

	// Should allow 100 requests without delay
	for i := 0; i < 100; i++ {
		if !rl.Allow(ip) {
			t.Errorf("Request %d should be allowed with burst of 100", i+1)
		}
	}

	// 101st should be denied
	if rl.Allow(ip) {
		t.Error("Request 101 should be denied (burst exhausted)")
	}
}

// TestIPRateLimiter_MultipleClientsIndependence tests that multiple clients have independent limiters
func TestIPRateLimiter_MultipleClientsIndependence(t *testing.T) {
	rl := NewIPRateLimiter(1, 2)

	ips := []string{"10.0.0.1", "10.0.0.2", "10.0.0.3", "10.0.0.4", "10.0.0.5"}

	// Exhaust burst for all IPs
	for _, ip := range ips {
		for i := 0; i < 2; i++ {
			if !rl.Allow(ip) {
				t.Errorf("IP %s request %d should be allowed (within burst)", ip, i+1)
			}
		}
	}

	// All IPs should be blocked on 3rd request
	for _, ip := range ips {
		if rl.Allow(ip) {
			t.Errorf("IP %s should be blocked (burst exhausted)", ip)
		}
	}

	// Wait for refill
	time.Sleep(1100 * time.Millisecond)

	// All IPs should allow 1 request independently
	for _, ip := range ips {
		if !rl.Allow(ip) {
			t.Errorf("IP %s should be allowed after refill", ip)
		}
		if rl.Allow(ip) {
			t.Errorf("IP %s should be blocked after 1 refill token consumed", ip)
		}
	}
}

// TestIPRateLimiter_LimiterCreation tests that limiters are created on first access
func TestIPRateLimiter_LimiterCreation(t *testing.T) {
	rl := NewIPRateLimiter(5, 5)

	// Before any requests, no limiters should exist
	rl.mu.RLock()
	initialCount := len(rl.limiters)
	rl.mu.RUnlock()

	if initialCount != 0 {
		t.Errorf("Expected 0 limiters initially, got %d", initialCount)
	}

	// After first request, limiter should be created
	ip := "192.168.2.1"
	rl.Allow(ip)

	rl.mu.RLock()
	if _, exists := rl.limiters[ip]; !exists {
		t.Error("Limiter should be created after first Allow() call")
	}
	rl.mu.RUnlock()
}

// TestIPRateLimiter_LastSeenUpdate tests that lastSeen is updated on each request
func TestIPRateLimiter_LastSeenUpdate(t *testing.T) {
	rl := NewIPRateLimiter(5, 5)
	ip := "192.168.3.1"

	// First request
	rl.Allow(ip)
	rl.mu.RLock()
	firstTime := rl.lastSeen[ip]
	rl.mu.RUnlock()

	// Wait a bit and make another request
	time.Sleep(50 * time.Millisecond)
	rl.Allow(ip)

	rl.mu.RLock()
	secondTime := rl.lastSeen[ip]
	rl.mu.RUnlock()

	if secondTime.Before(firstTime) {
		t.Error("lastSeen should be updated to current time on each request")
	}

	if secondTime.Equal(firstTime) {
		t.Error("lastSeen times should be different after waiting")
	}
}

// TestIPRateLimiter_CleanupMultipleStaleIPs tests cleanup with multiple expired limiters
func TestIPRateLimiter_CleanupMultipleStaleIPs(t *testing.T) {
	rl := NewIPRateLimiter(10, 10)
	rl.expiryDuration = 100 * time.Millisecond

	// Add multiple limiters
	ips := []string{"192.168.4.1", "192.168.4.2", "192.168.4.3"}
	for _, ip := range ips {
		rl.Allow(ip)
	}

	// Verify all exist
	rl.mu.RLock()
	if len(rl.limiters) != 3 {
		t.Fatalf("Expected 3 limiters, got %d", len(rl.limiters))
	}
	rl.mu.RUnlock()

	// Wait for expiry
	time.Sleep(150 * time.Millisecond)

	// Trigger cleanup
	rl.cleanup()

	// All should be removed
	rl.mu.RLock()
	if len(rl.limiters) != 0 {
		t.Errorf("Expected 0 limiters after cleanup, got %d", len(rl.limiters))
	}
	rl.mu.RUnlock()
}

// TestIPRateLimiter_CleanupPartialExpiry tests cleanup when only some limiters are expired
func TestIPRateLimiter_CleanupPartialExpiry(t *testing.T) {
	rl := NewIPRateLimiter(10, 10)
	rl.expiryDuration = 100 * time.Millisecond

	// Add first limiter
	ip1 := "192.168.5.1"
	rl.Allow(ip1)

	// Wait for partial expiry
	time.Sleep(80 * time.Millisecond)

	// Add second limiter (recent)
	ip2 := "192.168.5.2"
	rl.Allow(ip2)

	// Wait for first to expire
	time.Sleep(60 * time.Millisecond)

	// Trigger cleanup
	rl.cleanup()

	// ip1 should be removed, ip2 should remain
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	if _, exists := rl.limiters[ip1]; exists {
		t.Error("Expired limiter ip1 should be removed")
	}

	if _, exists := rl.limiters[ip2]; !exists {
		t.Error("Recent limiter ip2 should not be removed")
	}
}

// TestIPRateLimiter_CleanupOnlyAffectsExpired tests that cleanup doesn't affect recently used IPs
func TestIPRateLimiter_CleanupOnlyAffectsExpired(t *testing.T) {
	rl := NewIPRateLimiter(10, 10)
	rl.expiryDuration = 100 * time.Millisecond

	ip := "192.168.6.1"
	rl.Allow(ip)

	// Continuously access the limiter to keep it fresh
	time.Sleep(50 * time.Millisecond)
	rl.Allow(ip)

	time.Sleep(50 * time.Millisecond)
	rl.Allow(ip)

	// Trigger cleanup (limiter should not be expired)
	rl.cleanup()

	// Limiter should still exist
	rl.mu.RLock()
	if _, exists := rl.limiters[ip]; !exists {
		t.Error("Recently used limiter should not be removed by cleanup")
	}
	rl.mu.RUnlock()
}

// TestGetIP_RemoteAddrWithoutPort tests getIP with RemoteAddr that has no port
func TestGetIP_RemoteAddrWithoutPort(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1" // No port

	ip := getIP(req)
	if ip != "192.168.1.1" {
		t.Errorf("Expected IP 192.168.1.1, got %s", ip)
	}
}

// TestGetIP_EmptyForwardedFor tests getIP with empty X-Forwarded-For value
func TestGetIP_EmptyForwardedFor(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", "") // Empty header

	ip := getIP(req)
	if ip != "10.0.0.1" {
		t.Errorf("Expected fallback to RemoteAddr 10.0.0.1, got %s", ip)
	}
}

// TestGetIP_XForwardedForWithWhitespace tests X-Forwarded-For with various whitespace
func TestGetIP_XForwardedForWithWhitespace(t *testing.T) {
	tests := []struct {
		name       string
		forwarded  string
		expectedIP string
	}{
		{
			name:       "leading whitespace",
			forwarded:  "  203.0.113.1",
			expectedIP: "203.0.113.1",
		},
		{
			name:       "trailing whitespace",
			forwarded:  "203.0.113.1  ",
			expectedIP: "203.0.113.1",
		},
		{
			name:       "tabs and spaces",
			forwarded:  " \t 203.0.113.1 \t ",
			expectedIP: "203.0.113.1",
		},
		{
			name:       "multiple IPs with varying whitespace",
			forwarded:  "203.0.113.1 , 198.51.100.1,  192.0.2.1",
			expectedIP: "203.0.113.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = "10.0.0.1:12345"
			req.Header.Set("X-Forwarded-For", tt.forwarded)

			ip := getIP(req)
			if ip != tt.expectedIP {
				t.Errorf("Expected IP %s, got %s", tt.expectedIP, ip)
			}
		})
	}
}

// TestSplitCommaSeparated tests comma separation with edge cases
func TestSplitCommaSeparated(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single item",
			input:    "item1",
			expected: []string{"item1"},
		},
		{
			name:     "multiple items",
			input:    "item1,item2,item3",
			expected: []string{"item1", "item2", "item3"},
		},
		{
			name:     "trailing comma",
			input:    "item1,item2,",
			expected: []string{"item1", "item2"},
		},
		{
			name:     "leading comma",
			input:    ",item1,item2",
			expected: []string{"", "item1", "item2"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "only commas",
			input:    ",,,",
			expected: []string{"", "", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitCommaSeparated(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d items, got %d", len(tt.expected), len(result))
				return
			}
			for i, item := range result {
				if item != tt.expected[i] {
					t.Errorf("Item[%d]: expected %q, got %q", i, tt.expected[i], item)
				}
			}
		})
	}
}

// TestIPRateLimiter_RateLimitReset tests rate limit behavior over extended time
func TestIPRateLimiter_RateLimitReset(t *testing.T) {
	rl := NewIPRateLimiter(2, 2) // 2 req/sec with burst of 2
	ip := "192.168.7.1"

	// Use up burst
	rl.Allow(ip)
	rl.Allow(ip)

	// Should be blocked
	if rl.Allow(ip) {
		t.Error("Should be blocked after burst exhausted")
	}

	// Wait for 1 second (should get 2 tokens)
	time.Sleep(1100 * time.Millisecond)

	// Should allow 2 more requests
	if !rl.Allow(ip) {
		t.Error("Should allow request after 1 second")
	}
	if !rl.Allow(ip) {
		t.Error("Should allow second request within reset period")
	}

	// 3rd should be blocked
	if rl.Allow(ip) {
		t.Error("Should block 3rd request (burst of 2 only)")
	}

	// Wait for another second
	time.Sleep(1100 * time.Millisecond)

	// Should allow 2 more requests again
	if !rl.Allow(ip) {
		t.Error("Should allow request after second second")
	}
	if !rl.Allow(ip) {
		t.Error("Should allow second request in second second")
	}
}

// TestIPRateLimiter_ConcurrentAccess tests thread safety with concurrent requests
func TestIPRateLimiter_ConcurrentAccess(t *testing.T) {
	rl := NewIPRateLimiter(100, 50) // High limits to allow concurrent requests
	ip := "192.168.8.1"

	// Launch multiple goroutines accessing the same IP
	const numGoroutines = 10
	const requestsPerGoroutine = 5

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			for j := 0; j < requestsPerGoroutine; j++ {
				allowed := rl.Allow(ip)
				// We don't store results directly due to race conditions
				// but the test should complete without panicking
				_ = allowed
			}
		}(i)
	}

	// Wait for all goroutines to complete
	time.Sleep(500 * time.Millisecond)

	// Verify limiter still exists and is valid
	rl.mu.RLock()
	if _, exists := rl.limiters[ip]; !exists {
		t.Error("Limiter should still exist after concurrent access")
	}
	rl.mu.RUnlock()
}

// TestIPRateLimiter_ConcurrentMultipleIPs tests concurrent access from different IPs
func TestIPRateLimiter_ConcurrentMultipleIPs(t *testing.T) {
	rl := NewIPRateLimiter(100, 50)

	const numIPs = 5
	const requestsPerIP = 10

	for i := 0; i < numIPs; i++ {
		ip := fmt.Sprintf("192.168.100.%d", i)
		go func(ipAddr string) {
			for j := 0; j < requestsPerIP; j++ {
				rl.Allow(ipAddr)
				time.Sleep(1 * time.Millisecond)
			}
		}(ip)
	}

	// Wait for all goroutines
	time.Sleep(200 * time.Millisecond)

	// Verify all limiters were created
	rl.mu.RLock()
	if len(rl.limiters) != numIPs {
		t.Errorf("Expected %d limiters, got %d", numIPs, len(rl.limiters))
	}
	rl.mu.RUnlock()
}

// TestIPRateLimiter_Middleware_MultipleRequests tests middleware with multiple rapid requests
func TestIPRateLimiter_Middleware_MultipleRequests(t *testing.T) {
	rl := NewIPRateLimiter(5, 5)

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	server := httptest.NewServer(handler)
	defer server.Close()

	// Make 5 successful requests
	for i := 0; i < 5; i++ {
		resp, err := http.Get(server.URL)
		if err != nil {
			t.Fatalf("Request %d failed: %v", i+1, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Request %d: expected 200, got %d", i+1, resp.StatusCode)
		}
		resp.Body.Close()
	}

	// 6th request should be rate limited
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Request 6 failed: %v", err)
	}
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("Request 6: expected 429, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

// TestIPRateLimiter_Middleware_CustomIP tests middleware with custom IP headers
func TestIPRateLimiter_Middleware_CustomIP(t *testing.T) {
	rl := NewIPRateLimiter(2, 2)

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	server := httptest.NewServer(handler)
	defer server.Close()

	// Create requests with custom IPs via X-Forwarded-For
	client := &http.Client{}

	for i := 0; i < 2; i++ {
		req, err := http.NewRequest("GET", server.URL, nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("X-Forwarded-For", "203.0.113.1")

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request %d failed: %v", i+1, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Request %d: expected 200, got %d", i+1, resp.StatusCode)
		}
		resp.Body.Close()
	}

	// 3rd request should be rate limited
	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("X-Forwarded-For", "203.0.113.1")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request 3 failed: %v", err)
	}
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("Request 3: expected 429, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

// TestIPRateLimiter_VerySmallRate tests with very small rate (< 1 req/sec)
func TestIPRateLimiter_VerySmallRate(t *testing.T) {
	rl := NewIPRateLimiter(rate.Limit(0.5), 1) // 0.5 requests per second (1 req per 2 secs)
	ip := "192.168.9.1"

	// First request should be allowed (burst of 1)
	if !rl.Allow(ip) {
		t.Error("First request should be allowed")
	}

	// Second should be blocked
	if rl.Allow(ip) {
		t.Error("Second request should be blocked (burst of 1)")
	}

	// Wait 1 second (not enough for 0.5 req/sec)
	time.Sleep(1100 * time.Millisecond)
	if rl.Allow(ip) {
		t.Error("Request after 1 second should be blocked (need 2 seconds for 0.5 req/sec)")
	}

	// Wait another second (total 2 seconds)
	time.Sleep(1100 * time.Millisecond)
	if !rl.Allow(ip) {
		t.Error("Request after 2 seconds should be allowed")
	}
}
