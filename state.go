package circuitbreaker

// State represents the current state of the circuit breaker.
type State int

const (
	// Closed - normal operation, requests pass through
	Closed State = iota
	// Open - circuit tripped, requests fail immediately
	Open
	// HalfOpen - testing if service recovered
	HalfOpen
)

// String returns the string representation of the state.
func (s State) String() string {
	switch s {
	case Open:
		return "Open"
	case HalfOpen:
		return "HalfOpen"
	case Closed:
		return "Closed"
	default:
		return "Closed"
	}
}
