package circuitbreaker

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var errSimulated = errors.New("simulated failure")

// Helper: creates a circuit breaker with short timeout for testing
func newTestBreaker() *CircuitBreaker {
	return New(Config{
		Name:             "test",
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          100 * time.Millisecond,
	})
}

// Helper: function that always succeeds
func successFn() (any, error) {
	return "ok", nil
}

// Helper: function that always fails
func failFn() (any, error) {
	return nil, errSimulated
}

func TestNew(t *testing.T) {
	cb := newTestBreaker()

	if cb == nil {
		t.Fatal("New returned nil")
	}

	if cb.State() != Closed {
		t.Errorf("expected initial state Closed, got %v", cb.State())
	}

	if cb.failures != 0 {
		t.Errorf("expected 0 failures, got %d", cb.failures)
	}

	if cb.successes != 0 {
		t.Errorf("expected 0 successes, got %d", cb.successes)
	}
}

func TestExecute_Success(t *testing.T) {
	cb := newTestBreaker()

	result, err := cb.Execute(successFn)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if result != "ok" {
		t.Errorf("expected 'ok', got %v", result)
	}

	if cb.State() != Closed {
		t.Errorf("expected state Closed after success, got %v", cb.State())
	}
}

func TestExecute_Failure(t *testing.T) {
	cb := newTestBreaker()

	result, err := cb.Execute(failFn)

	if err != errSimulated {
		t.Errorf("expected errSimulated, got %v", err)
	}

	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}

	if cb.failures != 1 {
		t.Errorf("expected 1 failure, got %d", cb.failures)
	}
}

func TestStateTransition_ClosedToOpen(t *testing.T) {
	cb := newTestBreaker() // FailureThreshold = 3

	// Cause 3 failures to trip the breaker
	for i := 0; i < 3; i++ {
		cb.Execute(failFn)
	}

	if cb.State() != Open {
		t.Errorf("expected state Open after %d failures, got %v", 3, cb.State())
	}

	// Next request should be rejected immediately
	_, err := cb.Execute(successFn)
	if err != ErrCircuitOpen {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestStateTransition_OpenToHalfOpen(t *testing.T) {
	cb := newTestBreaker() // Timeout = 100ms

	// Trip the breaker
	for i := 0; i < 3; i++ {
		cb.Execute(failFn)
	}

	if cb.State() != Open {
		t.Fatalf("expected state Open, got %v", cb.State())
	}

	// Wait for timeout to expire
	time.Sleep(150 * time.Millisecond)

	// Next request should transition to HalfOpen and go through
	_, err := cb.Execute(successFn)
	if err == ErrCircuitOpen {
		t.Error("expected request to go through after timeout, got ErrCircuitOpen")
	}

	if cb.State() != HalfOpen {
		t.Errorf("expected state HalfOpen after timeout, got %v", cb.State())
	}
}

func TestStateTransition_HalfOpenToClosed(t *testing.T) {
	cb := newTestBreaker() // SuccessThreshold = 2

	// Trip the breaker
	for i := 0; i < 3; i++ {
		cb.Execute(failFn)
	}

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// First success - should transition to HalfOpen
	cb.Execute(successFn)

	// Second success - should close the circuit
	cb.Execute(successFn)

	if cb.State() != Closed {
		t.Errorf("expected state Closed after %d successes in HalfOpen, got %v", 2, cb.State())
	}
}

func TestStateTransition_HalfOpenToOpen(t *testing.T) {
	cb := newTestBreaker()

	// Trip the breaker
	for i := 0; i < 3; i++ {
		cb.Execute(failFn)
	}

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// First request transitions to HalfOpen
	cb.Execute(successFn)

	if cb.State() != HalfOpen {
		t.Fatalf("expected HalfOpen, got %v", cb.State())
	}

	// Failure in HalfOpen should reopen immediately
	cb.Execute(failFn)

	if cb.State() != Open {
		t.Errorf("expected state Open after failure in HalfOpen, got %v", cb.State())
	}
}

func TestReset(t *testing.T) {
	cb := newTestBreaker()

	// Trip the breaker
	for i := 0; i < 3; i++ {
		cb.Execute(failFn)
	}

	if cb.State() != Open {
		t.Fatalf("expected Open, got %v", cb.State())
	}

	// Reset
	cb.Reset()

	if cb.State() != Closed {
		t.Errorf("expected Closed after Reset, got %v", cb.State())
	}

	if cb.failures != 0 {
		t.Errorf("expected 0 failures after Reset, got %d", cb.failures)
	}

	if cb.successes != 0 {
		t.Errorf("expected 0 successes after Reset, got %d", cb.successes)
	}

	// Should work normally after reset
	_, err := cb.Execute(successFn)
	if err != nil {
		t.Errorf("expected success after Reset, got %v", err)
	}
}

func TestSuccessResetsFailureCount(t *testing.T) {
	cb := newTestBreaker() // FailureThreshold = 3

	// 2 failures (not enough to trip)
	cb.Execute(failFn)
	cb.Execute(failFn)

	if cb.failures != 2 {
		t.Fatalf("expected 2 failures, got %d", cb.failures)
	}

	// 1 success should reset the failure count
	cb.Execute(successFn)

	if cb.failures != 0 {
		t.Errorf("expected failures reset to 0 after success, got %d", cb.failures)
	}

	// Now need 3 more failures to trip
	cb.Execute(failFn)
	cb.Execute(failFn)

	if cb.State() != Closed {
		t.Errorf("expected still Closed (only 2 failures), got %v", cb.State())
	}
}

func TestConcurrency(t *testing.T) {
	cb := newTestBreaker()

	var wg sync.WaitGroup
	var successCount atomic.Int32
	var errorCount atomic.Int32

	// Launch 100 goroutines
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := cb.Execute(successFn)
			if err != nil {
				errorCount.Add(1)
			} else {
				successCount.Add(1)
			}
		}()
	}

	wg.Wait()

	// All should succeed since we only called successFn
	if successCount.Load() != 100 {
		t.Errorf("expected 100 successes, got %d (errors: %d)", successCount.Load(), errorCount.Load())
	}

	if cb.State() != Closed {
		t.Errorf("expected Closed after all successes, got %v", cb.State())
	}
}

func TestConcurrency_WithFailures(t *testing.T) {
	cb := New(Config{
		Name:             "test",
		FailureThreshold: 5,
		SuccessThreshold: 2,
		Timeout:          100 * time.Millisecond,
	})

	var wg sync.WaitGroup
	var circuitOpenCount atomic.Int32

	// Launch 50 goroutines that all fail
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := cb.Execute(failFn)
			if err == ErrCircuitOpen {
				circuitOpenCount.Add(1)
			}
		}()
	}

	wg.Wait()

	// Circuit should be open
	if cb.State() != Open {
		t.Errorf("expected Open after many failures, got %v", cb.State())
	}

	// Some requests should have been rejected with ErrCircuitOpen
	if circuitOpenCount.Load() == 0 {
		t.Log("Note: no requests were rejected - all may have executed before circuit opened")
	}
}

func TestState_ReturnsCurrentState(t *testing.T) {
	cb := newTestBreaker()

	if cb.State() != Closed {
		t.Errorf("expected Closed, got %v", cb.State())
	}

	// Trip it
	for i := 0; i < 3; i++ {
		cb.Execute(failFn)
	}

	if cb.State() != Open {
		t.Errorf("expected Open, got %v", cb.State())
	}
}
