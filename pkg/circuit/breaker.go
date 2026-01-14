package circuit

import (
	"fmt"
	"sync"
	"time"
)

// State represents the circuit breaker state
type State int

const (
	// StateClosed allows requests to pass through
	StateClosed State = iota
	// StateOpen blocks requests
	StateOpen
	// StateHalfOpen allows limited requests to test if service recovered
	StateHalfOpen
)

// String returns the string representation of the state
func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	mu           sync.RWMutex
	state        State
	failures     int
	successes    int
	lastFailure  time.Time
	lastStateChange time.Time

	maxFailures int
	timeout     time.Duration
	halfOpenMax int // max successes needed in half-open to close
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(maxFailures int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:        StateClosed,
		maxFailures:  maxFailures,
		timeout:      timeout,
		halfOpenMax:  2, // Need 2 successes to close
		lastStateChange: time.Now(),
	}
}

// Call executes the function with circuit breaker protection
func (cb *CircuitBreaker) Call(fn func() error) error {
	cb.mu.Lock()

	// Check if we should transition from open to half-open
	if cb.state == StateOpen {
		if time.Since(cb.lastFailure) > cb.timeout {
			cb.state = StateHalfOpen
			cb.successes = 0
			cb.failures = 0
			cb.lastStateChange = time.Now()
		} else {
			cb.mu.Unlock()
			return fmt.Errorf("circuit breaker open (will retry in %v)",
				cb.timeout-time.Since(cb.lastFailure))
		}
	}

	// In half-open state, allow limited requests
	if cb.state == StateHalfOpen {
		if cb.failures > 0 {
			// Already had a failure in half-open, stay open
			cb.mu.Unlock()
			return fmt.Errorf("circuit breaker half-open but failing")
		}
	}

	cb.mu.Unlock()

	// Execute the function
	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.failures++
		cb.lastFailure = time.Now()

		if cb.state == StateHalfOpen {
			// Failure in half-open goes back to open
			cb.state = StateOpen
			cb.lastStateChange = time.Now()
		} else if cb.failures >= cb.maxFailures {
			// Too many failures, open the circuit
			cb.state = StateOpen
			cb.lastStateChange = time.Now()
		}

		return err
	}

	// Success
	if cb.state == StateHalfOpen {
		cb.successes++
		if cb.successes >= cb.halfOpenMax {
			// Enough successes, close the circuit
			cb.state = StateClosed
			cb.failures = 0
			cb.successes = 0
			cb.lastStateChange = time.Now()
		}
	} else if cb.state == StateClosed {
		// Reset failure count on success
		cb.failures = 0
	}

	return nil
}

// State returns the current circuit breaker state
func (cb *CircuitBreaker) GetState() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetFailures returns the current failure count
func (cb *CircuitBreaker) GetFailures() int {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.failures
}

// Reset manually resets the circuit breaker
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = StateClosed
	cb.failures = 0
	cb.successes = 0
	cb.lastStateChange = time.Now()
}
