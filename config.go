package circuitbreaker

import "time"

// Config holds the circuit breaker configuration.
type Config struct {
	// Name identifies this circuit breaker (for logging/metrics)
	Name string

	// FailureThreshold is the number of consecutive failures before opening
	FailureThreshold int

	// SuccessThreshold is the number of successes in half-open state to close
	SuccessThreshold int

	// Timeout is how long to stay open before transitioning to half-open
	Timeout time.Duration
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		Name:             "default",
		FailureThreshold: 3,
		SuccessThreshold: 5,
		Timeout:          10 * time.Second,
	}
}

// Validate checks that the config is valid.
func (c Config) Validate() error {
	// TODO: implement
	return nil
}
