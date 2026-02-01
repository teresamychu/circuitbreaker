# Circuit Breaker

A Go implementation of the circuit breaker pattern for fault tolerance in distributed systems.

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
        Name:             "api-service",
        FailureThreshold: 5,        // Open after 5 consecutive failures
        SuccessThreshold: 2,        // Close after 2 successes in half-open
        Timeout:          30*time.Second, // Wait 30s before trying again
    })

    result, err := cb.Execute(func() (any, error) {
        resp, err := http.Get("https://api.example.com/data")
        if err != nil {
            return nil, err
        }
        return resp, nil
    })

    if err == circuitbreaker.ErrCircuitOpen {
        fmt.Println("Service unavailable, circuit is open")
        return
    }

    if err != nil {
        fmt.Println("Request failed:", err)
        return
    }

    fmt.Println("Success:", result)
}
```

## Circuit Breaker Pattern

```
     ┌─────────┐     failures >= threshold     ┌────────┐
     │ CLOSED  │ ─────────────────────────────▶│  OPEN  │
     └─────────┘                               └────────┘
          ▲                                         │
          │                                    timeout expires
          │         success                         │
          │    ◀─────────────                       ▼
          │                  │               ┌───────────┐
          └──────────────────┴───────────────│ HALF-OPEN │
                          failure            └───────────┘
                    (back to OPEN)
```

### States

| State | Description |
|-------|-------------|
| **Closed** | Normal operation. Requests pass through. Failures are counted. |
| **Open** | Circuit tripped. Requests fail immediately with `ErrCircuitOpen`. |
| **Half-Open** | Testing recovery. Limited requests allowed. Success closes, failure reopens. |

## API

### `New(config Config) *CircuitBreaker`

Creates a new circuit breaker.

### `Execute(fn func() (any, error)) (any, error)`

Runs the function with circuit breaker protection. Returns `ErrCircuitOpen` if the circuit is open.

### `State() State`

Returns the current state (`Closed`, `Open`, or `HalfOpen`).

### `Reset()`

Manually resets the circuit breaker to closed state.

## Configuration

| Option | Description | Default |
|--------|-------------|---------|
| `Name` | Identifier for logging/metrics | `""` |
| `FailureThreshold` | Consecutive failures before opening | `5` |
| `SuccessThreshold` | Successes in half-open to close | `2` |
| `Timeout` | Duration in open before half-open | `30s` |

## License

MIT
