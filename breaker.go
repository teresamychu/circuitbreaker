// Package circuitbreaker implements the circuit breaker pattern for fault tolerance.
//
// A circuit breaker prevents cascading failures in distributed systems by
// failing fast when a service is unhealthy, giving it time to recover.
//
// Example usage:
//
//	cb := circuitbreaker.New(circuitbreaker.Config{
//	    Name:             "my-service",
//	    FailureThreshold: 5,
//	    SuccessThreshold: 2,
//	    Timeout:          30 * time.Second,
//	})
//
//	result, err := cb.Execute(func() (any, error) {
//	    return http.Get("https://api.example.com/data")
//	})
package circuitbreaker

import (
	"errors"
	"sync"
	"time"
)

// ErrCircuitOpen is returned when the circuit is open and requests are rejected.
var ErrCircuitOpen = errors.New("circuit breaker is open")
var ErrHalfOpen = errors.New("circuit breaker is half-open and has reached the threshold")

// CircuitBreaker implements the circuit breaker pattern.
type CircuitBreaker struct {
	config Config

	mu              sync.RWMutex
	state           State
	failures        int
	successes       int
	lastFailureTime time.Time
	lastStateChange time.Time
}

// New creates a new circuit breaker with the given config.
func New(config Config) *CircuitBreaker {
	c := CircuitBreaker{
		config: config,
	}
	return &c

}

// Execute runs the given function with circuit breaker protection.
// Returns ErrCircuitOpen if the circuit is open.
func (cb *CircuitBreaker) Execute(fn func() (any, error)) (any, error) {
	cb.mu.Lock()
	//check status of circuit breaker
	if cb.state == Open {
		return nil, ErrCircuitOpen
	}
	if cb.state == HalfOpen {
		if cb.successes == cb.config.SuccessThreshold {
			return nil, ErrHalfOpen
		}
	}
	defer cb.mu.Unlock()
	result, err := fn()

	//process result in circuit breaker. update circuit breaker state.
	return result, err

}

// State returns the current state of the circuit breaker.
func (cb *CircuitBreaker) State() State {
	return cb.state
}

// Reset manually resets the circuit breaker to closed state.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures = 0
	cb.lastFailureTime = time.Time{}
	cb.lastStateChange = time.Time{}
	cb.successes = 0
	cb.state = Closed
}
