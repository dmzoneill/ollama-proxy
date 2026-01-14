package middleware

import (
	"context"
	"errors"
	"testing"

	"github.com/daoneill/ollama-proxy/pkg/logging"
)

func TestWithRequestID(t *testing.T) {
	ctx := context.Background()

	// Add request ID to context
	ctxWithID := WithRequestID(ctx)

	// Verify request ID was added
	requestID := GetRequestID(ctxWithID)
	if requestID == "" {
		t.Error("Expected non-empty request ID")
	}

	// Verify request ID is a valid UUID format (basic check)
	if len(requestID) != 36 {
		t.Errorf("Expected request ID length of 36, got %d", len(requestID))
	}
}

func TestGetRequestID(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() context.Context
		wantEmpty bool
	}{
		{
			name: "context with request ID",
			setup: func() context.Context {
				return WithRequestID(context.Background())
			},
			wantEmpty: false,
		},
		{
			name: "context without request ID",
			setup: func() context.Context {
				return context.Background()
			},
			wantEmpty: true,
		},
		{
			name: "context with wrong type value",
			setup: func() context.Context {
				return context.WithValue(context.Background(), RequestIDKey, 12345)
			},
			wantEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setup()
			requestID := GetRequestID(ctx)

			if tt.wantEmpty && requestID != "" {
				t.Errorf("Expected empty request ID, got %s", requestID)
			}
			if !tt.wantEmpty && requestID == "" {
				t.Error("Expected non-empty request ID")
			}
		})
	}
}

func TestRequestIDUniqueness(t *testing.T) {
	ctx := context.Background()

	// Create multiple request IDs
	ctx1 := WithRequestID(ctx)
	ctx2 := WithRequestID(ctx)
	ctx3 := WithRequestID(ctx)

	id1 := GetRequestID(ctx1)
	id2 := GetRequestID(ctx2)
	id3 := GetRequestID(ctx3)

	// Verify all IDs are unique
	if id1 == id2 || id1 == id3 || id2 == id3 {
		t.Error("Request IDs should be unique")
	}
}

func TestRecoverPanic(t *testing.T) {
	// Initialize logger for testing
	if err := logging.InitLogger("info", false); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	tests := []struct {
		name       string
		handler    func() error
		wantErr    bool
		wantPanic  bool
	}{
		{
			name: "handler succeeds",
			handler: func() error {
				return nil
			},
			wantErr:   false,
			wantPanic: false,
		},
		{
			name: "handler returns error",
			handler: func() error {
				return errors.New("test error")
			},
			wantErr:   true,
			wantPanic: false,
		},
		{
			name: "handler panics with string",
			handler: func() error {
				panic("test panic")
			},
			wantErr:   true,
			wantPanic: true,
		},
		{
			name: "handler panics with error",
			handler: func() error {
				panic(errors.New("panic error"))
			},
			wantErr:   true,
			wantPanic: true,
		},
		{
			name: "handler panics with nil",
			handler: func() error {
				panic(nil)
			},
			wantErr:   true,
			wantPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := WithRequestID(context.Background())

			err := RecoverPanic(ctx, tt.handler)

			if (err != nil) != tt.wantErr {
				t.Errorf("RecoverPanic() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantPanic && err != nil {
				// Verify error message indicates panic was recovered
				expectedMsg := "internal server error: panic recovered"
				if err.Error() != expectedMsg {
					t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
				}
			}
		})
	}
}

func TestRecoverPanicWithoutRequestID(t *testing.T) {
	// Initialize logger for testing
	if err := logging.InitLogger("info", false); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	ctx := context.Background() // No request ID

	err := RecoverPanic(ctx, func() error {
		panic("test panic without request ID")
	})

	if err == nil {
		t.Error("Expected error from panic recovery")
	}

	expectedMsg := "internal server error: panic recovered"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestRecoverPanicPreservesOriginalError(t *testing.T) {
	// Initialize logger for testing
	if err := logging.InitLogger("info", false); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	ctx := WithRequestID(context.Background())
	originalErr := errors.New("original error")

	err := RecoverPanic(ctx, func() error {
		return originalErr
	})

	if err != originalErr {
		t.Errorf("Expected original error to be preserved, got %v", err)
	}
}

func TestContextKeyType(t *testing.T) {
	// Verify the context key is of the correct type
	var key interface{} = RequestIDKey
	if _, ok := key.(contextKey); !ok {
		t.Error("RequestIDKey should be of type contextKey")
	}
}
