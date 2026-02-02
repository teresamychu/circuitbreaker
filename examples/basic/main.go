// Basic example demonstrating circuit breaker with a flaky service
package main

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/teresamychu/circuitbreaker"
)

var errServiceDown = errors.New("service unavailable")

// Simulates a flaky service that fails 70% of the time
func flakyService() (any, error) {
	if rand.Float32() < 0.7 {
		return nil, errServiceDown
	}
	return "success!", nil
}

func main() {
	cb := circuitbreaker.New(circuitbreaker.Config{
		Name:             "flaky-service",
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          2 * time.Second,
	})

	fmt.Println("Circuit Breaker Demo - Flaky Service")
	fmt.Println("=====================================")
	fmt.Printf("Config: FailureThreshold=%d, SuccessThreshold=%d, Timeout=%s\n\n",
		3, 2, 2*time.Second)

	// Make 20 requests over time
	for i := 1; i <= 20; i++ {
		result, err := cb.Execute(flakyService)

		state := cb.State()
		if err == circuitbreaker.ErrCircuitOpen {
			fmt.Printf("Request %2d: [%s] REJECTED - circuit is open\n", i, state)
		} else if err != nil {
			fmt.Printf("Request %2d: [%s] FAILED - %v\n", i, state, err)
		} else {
			fmt.Printf("Request %2d: [%s] SUCCESS - %v\n", i, state, result)
		}

		time.Sleep(300 * time.Millisecond)
	}
}
