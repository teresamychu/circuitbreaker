# Circuit Breaker

A Go implementation of the circuit breaker pattern for building resilient applications.

## Installation

```bash
go get github.com/teresamychu/circuitbreaker
```

## Usage

```go
package main

import (
    "fmt"
    "net/http"
    "time"

    "github.com/teresamychu/circuitbreaker"
)

func main() {
    cb := circuitbreaker.New(circuitbreaker.Config{
        Name:             "my-service",
        FailureThreshold: 5,        // Open after 5 consecutive failures
        SuccessThreshold: 2,        // Close after 2 successes in half-open
        Timeout:          30*time.Second,
    })

    result, err := cb.Execute(func() (any, error) {
        resp, err := http.Get("https://api.example.com/data")
        if err != nil {
            return nil, err
        }
        return resp, nil
    })

    if err == circuitbreaker.ErrCircuitOpen {
        fmt.Println("Circuit is open - failing fast")
        return
    }
}
```

## How It Works

```
     ┌─────────┐  failures >= threshold  ┌────────┐
     │ CLOSED  │ ───────────────────────>│  OPEN  │
     └─────────┘                         └────────┘
          ^                                   │
          │                              timeout expires
          │      successes >= threshold       │
          │     <─────────────────────        v
          │                           ┌───────────┐
          └───────────────────────────│ HALF-OPEN │
                    failure ──────────└───────────┘
                       │                    │
                       v                    │
                    ┌────────┐              │
                    │  OPEN  │<─────────────┘
                    └────────┘
```

| State | Behavior |
|-------|----------|
| **Closed** | Normal operation. Requests pass through. Consecutive failures are counted. |
| **Open** | Requests fail immediately with `ErrCircuitOpen`. After timeout, transitions to Half-Open. |
| **Half-Open** | Probe requests are allowed through. Success closes the circuit; failure reopens it. |

## Configuration

| Option | Description | Default |
|--------|-------------|---------|
| `Name` | Identifier for the circuit breaker | `"default"` |
| `FailureThreshold` | Consecutive failures before opening | `3` |
| `SuccessThreshold` | Successes in half-open to close | `5` |
| `Timeout` | Time in open state before half-open | `10s` |

## API

### `New(config Config) *CircuitBreaker`
Creates a new circuit breaker with the given configuration.

### `Execute(fn func() (any, error)) (any, error)`
Executes the function with circuit breaker protection. Returns `ErrCircuitOpen` if the circuit is open.

### `State() State`
Returns the current state: `Closed`, `Open`, or `HalfOpen`.

### `Reset()`
Manually resets the circuit breaker to closed state.

## Examples

Run the examples to see the circuit breaker in action:

```bash
# Basic example with simulated flaky service
go run ./examples/basic

# HTTP example with real API calls
go run ./examples/http

# HTTP server with circuit breaker protection
go run ./examples/server
# Then: curl http://localhost:8080/api/data
```

## License

MIT
