package circuit

import (
	"errors"
	"testing"
	"time"
)

func TestCircuitBreaker_InitialState(t *testing.T) {
	cb := NewCircuitBreaker(3, 1*time.Second)
	if cb.GetState() != StateClosed {
		t.Errorf("Expected initial state to be Closed, got %v", cb.GetState())
	}
}

func TestCircuitBreaker_SuccessfulCalls(t *testing.T) {
	cb := NewCircuitBreaker(3, 1*time.Second)

	for i := 0; i < 10; i++ {
		err := cb.Call(func() error {
			return nil
		})
		if err != nil {
			t.Errorf("Expected success, got error: %v", err)
		}
	}

	if cb.GetState() != StateClosed {
		t.Errorf("Expected state to remain Closed after successes, got %v", cb.GetState())
	}
}

func TestCircuitBreaker_OpenAfterFailures(t *testing.T) {
	cb := NewCircuitBreaker(3, 1*time.Second)

	testErr := errors.New("test error")

	// Fail 3 times
	for i := 0; i < 3; i++ {
		cb.Call(func() error {
			return testErr
		})
	}

	if cb.GetState() != StateOpen {
		t.Errorf("Expected state to be Open after max failures, got %v", cb.GetState())
	}

	// Should reject calls while open
	err := cb.Call(func() error {
		t.Error("Function should not be called when circuit is open")
		return nil
	})

	if err == nil {
		t.Error("Expected error when circuit is open")
	}
}

func TestCircuitBreaker_HalfOpenAfterTimeout(t *testing.T) {
	cb := NewCircuitBreaker(2, 100*time.Millisecond)

	testErr := errors.New("test error")

	// Fail to open the circuit
	for i := 0; i < 2; i++ {
		cb.Call(func() error {
			return testErr
		})
	}

	if cb.GetState() != StateOpen {
		t.Fatalf("Expected state to be Open, got %v", cb.GetState())
	}

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Next call should transition to half-open
	called := false
	cb.Call(func() error {
		called = true
		return nil // Success
	})

	if !called {
		t.Error("Function should be called in half-open state")
	}
}

func TestCircuitBreaker_CloseAfterHalfOpenSuccesses(t *testing.T) {
	cb := NewCircuitBreaker(2, 100*time.Millisecond)

	testErr := errors.New("test error")

	// Open the circuit
	for i := 0; i < 2; i++ {
		cb.Call(func() error {
			return testErr
		})
	}

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Succeed twice in half-open to close the circuit
	for i := 0; i < 2; i++ {
		err := cb.Call(func() error {
			return nil
		})
		if err != nil {
			t.Errorf("Expected success in half-open, got error: %v", err)
		}
	}

	if cb.GetState() != StateClosed {
		t.Errorf("Expected state to be Closed after half-open successes, got %v", cb.GetState())
	}
}

func TestCircuitBreaker_FailureInHalfOpenGoesBackToOpen(t *testing.T) {
	cb := NewCircuitBreaker(2, 100*time.Millisecond)

	testErr := errors.New("test error")

	// Open the circuit
	for i := 0; i < 2; i++ {
		cb.Call(func() error {
			return testErr
		})
	}

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Fail in half-open state
	cb.Call(func() error {
		return testErr
	})

	if cb.GetState() != StateOpen {
		t.Errorf("Expected state to be Open after half-open failure, got %v", cb.GetState())
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := NewCircuitBreaker(2, 1*time.Second)

	testErr := errors.New("test error")

	// Open the circuit
	for i := 0; i < 2; i++ {
		cb.Call(func() error {
			return testErr
		})
	}

	if cb.GetState() != StateOpen {
		t.Fatalf("Expected state to be Open, got %v", cb.GetState())
	}

	// Reset
	cb.Reset()

	if cb.GetState() != StateClosed {
		t.Errorf("Expected state to be Closed after reset, got %v", cb.GetState())
	}

	if cb.GetFailures() != 0 {
		t.Errorf("Expected failures to be 0 after reset, got %d", cb.GetFailures())
	}
}

func TestCircuitBreaker_StateString(t *testing.T) {
	tests := []struct {
		state    State
		expected string
	}{
		{StateClosed, "closed"},
		{StateOpen, "open"},
		{StateHalfOpen, "half-open"},
		{State(99), "unknown"}, // Test default case
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.expected {
			t.Errorf("State.String() = %v, want %v", got, tt.expected)
		}
	}
}

