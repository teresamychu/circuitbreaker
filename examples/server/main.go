// Server example - HTTP server that uses circuit breaker to protect downstream calls
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/teresamychu/circuitbreaker"
)

var cb *circuitbreaker.CircuitBreaker

// Simulates calling a downstream service
func callDownstreamService() (any, error) {
	// Simulate latency
	time.Sleep(time.Duration(50+rand.Intn(100)) * time.Millisecond)

	// 40% chance of failure
	if rand.Float32() < 0.4 {
		return nil, fmt.Errorf("downstream service error")
	}
	return map[string]string{"data": "from downstream"}, nil
}

func apiHandler(w http.ResponseWriter, r *http.Request) {
	result, err := cb.Execute(callDownstreamService)

	w.Header().Set("Content-Type", "application/json")

	response := map[string]any{
		"circuit_state": cb.State().String(),
	}

	if errors.Is(err, circuitbreaker.ErrCircuitOpen) {
		w.WriteHeader(http.StatusServiceUnavailable)
		response["error"] = "service temporarily unavailable"
		response["message"] = "circuit breaker is open, please retry later"
	} else if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		response["error"] = err.Error()
	} else {
		w.WriteHeader(http.StatusOK)
		response["data"] = result
	}

	json.NewEncoder(w).Encode(response)
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"circuit_state": cb.State().String(),
	})
}

func main() {
	cb = circuitbreaker.New(circuitbreaker.Config{
		Name:             "downstream-api",
		FailureThreshold: 5,
		SuccessThreshold: 3,
		Timeout:          10 * time.Second,
	})

	http.HandleFunc("/api/data", apiHandler)
	http.HandleFunc("/status", statusHandler)

	fmt.Println("Server running on http://localhost:8080")
	fmt.Println("Endpoints:")
	fmt.Println("  GET /api/data - calls downstream service (protected by circuit breaker)")
	fmt.Println("  GET /status   - shows circuit breaker state")
	fmt.Println("\nTry: curl http://localhost:8080/api/data")
	fmt.Println("     curl http://localhost:8080/status")

	log.Fatal(http.ListenAndServe(":8080", nil))
}
