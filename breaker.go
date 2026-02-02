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
var ErrFailedChecks = errors.New("failed pre-request checks")

// CircuitBreaker implements the circuit breaker pattern.
type CircuitBreaker struct {
	config Config

	mu sync.RWMutex
	// State of the circuit breaker: open, closed or half-open
	state State
	// Count for number of failures in current state.
	failures int
	//Count for number of successes in current state.
	successes int
	//The last failed request timestamp
	lastFailureTime time.Time
	//The last state change timestamp.
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
func (cb *CircuitBreaker) Execute(request func() (any, error)) (any, error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	canExecute := cb.canExecuteRequest()
	if !canExecute {
		return nil, ErrCircuitOpen
	}
	result, err := request()
	//process result in circuit breaker. update circuit breaker state.
	cb.afterRequestUpdates(result, err)
	return result, err
}

func (cb *CircuitBreaker) afterRequestUpdates(result any, err error) {
	if err != nil {
		//update circuit breaker with failure
		if cb.state == HalfOpen {
			cb.state = Open
			cb.lastStateChange = time.Now()
		}
		cb.failures++
		if cb.failures >= cb.config.FailureThreshold {
			//last request hit the threshold, open the circuit.
			cb.state = Open
			cb.lastStateChange = time.Now()
		}

		return
	}
	//update circuit breaker with success
	cb.failures = 0
	cb.successes++

	if (cb.successes >= cb.config.SuccessThreshold) && cb.state == HalfOpen {
		cb.state = Closed
	}
	return

}

// check before running the request to see where the circuit breaker is at.
// return true if checks succeed and request can be passed through, false if not.
func (cb *CircuitBreaker) canExecuteRequest() bool {
	//Before the request...

	//check status of circuit breaker
	if cb.state == Open {
		//if its been longer than the timeout since the last time the circuit breaker had changed, then return true.
		if time.Since(cb.lastStateChange) >= cb.config.Timeout {
			cb.state = HalfOpen
			return true
		}
		return false
	}
	if cb.state == HalfOpen {
		if time.Since(cb.lastFailureTime) >= cb.config.Timeout {
			return true
		}

	}
	if cb.state == Closed {
		return true
	}
	// if we get here something has gone very wrong.

	return false
}

func (cb *CircuitBreaker) processFailure() {
	cb.failures++
	cb.lastFailureTime = time.Now()
	cb.lastStateChange = time.Now()

	// if we have reached or somehow gone over our failure threshold,
	// open the circuit.
	if cb.failures >= cb.config.FailureThreshold {
		cb.state = Open
	}
}

func (cb *CircuitBreaker) onStateChange() {
	cb.Reset()
}

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
