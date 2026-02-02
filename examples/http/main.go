// HTTP example demonstrating circuit breaker with real HTTP calls
package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/teresamychu/circuitbreaker"
)

func main() {
	cb := circuitbreaker.New(circuitbreaker.Config{
		Name:             "http-api",
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          5 * time.Second,
	})

	// URLs to test - mix of valid and invalid
	urls := []string{
		"https://httpbin.org/status/200",  // Success
		"https://httpbin.org/status/200",  // Success
		"https://httpbin.org/status/500",  // Server error
		"https://httpbin.org/status/500",  // Server error
		"https://httpbin.org/status/500",  // Server error - should trip breaker
		"https://httpbin.org/status/200",  // Should be rejected (circuit open)
		"https://httpbin.org/status/200",  // Should be rejected (circuit open)
	}

	fmt.Println("Circuit Breaker Demo - HTTP Calls")
	fmt.Println("==================================\n")

	client := &http.Client{Timeout: 5 * time.Second}

	for i, url := range urls {
		result, err := cb.Execute(func() (any, error) {
			resp, err := client.Get(url)
			if err != nil {
				return nil, err
			}
			defer resp.Body.Close()

			if resp.StatusCode >= 500 {
				return nil, fmt.Errorf("server error: %d", resp.StatusCode)
			}
			return resp.StatusCode, nil
		})

		state := cb.State()
		if err == circuitbreaker.ErrCircuitOpen {
			fmt.Printf("Request %d: [%s] REJECTED - %s\n", i+1, state, url)
		} else if err != nil {
			fmt.Printf("Request %d: [%s] FAILED - %v\n", i+1, state, err)
		} else {
			fmt.Printf("Request %d: [%s] SUCCESS - status %v\n", i+1, state, result)
		}

		time.Sleep(500 * time.Millisecond)
	}

	// Wait for circuit to transition to half-open
	fmt.Println("\nWaiting 6 seconds for circuit timeout...")
	time.Sleep(6 * time.Second)

	// Try again - should be in half-open
	fmt.Println("\nRetrying after timeout:")
	result, err := cb.Execute(func() (any, error) {
		resp, err := client.Get("https://httpbin.org/status/200")
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		return resp.StatusCode, nil
	})

	state := cb.State()
	if err != nil {
		fmt.Printf("Request: [%s] FAILED - %v\n", state, err)
	} else {
		fmt.Printf("Request: [%s] SUCCESS - status %v\n", state, result)
	}
}
