package container

import (
	"fmt"
	"namor/pkg/errors"
	"namor/pkg/utils"
	"sync"
	"time"
)

type CircuitBreakerError struct {
	*errors.BaseError
	FromState CircuitBreakerState
	ToState   CircuitBreakerState
}

func NewCircuitBreakerStateChangeError(from, to CircuitBreakerState, message string) *CircuitBreakerError {
	return &CircuitBreakerError{
		BaseError: &errors.BaseError{
			Message: message,
			Code:    "CIRCUIT_BREAKER_STATE_CHANGE",
			Details: map[string]interface{}{
				"from": from.String(),
				"to":   to.String(),
			},
		},
		FromState: from,
		ToState:   to,
	}
}

func NewCircuitBreakerOpenError(failureCount int, retryAfter time.Duration) *CircuitBreakerError {
	return &CircuitBreakerError{
		BaseError: &errors.BaseError{
			Message: fmt.Sprintf("circuit breaker is OPEN (fails: %d, retry after: %v)", failureCount, retryAfter),
			Code:    "CIRCUIT_BREAKER_OPEN",
			Details: map[string]interface{}{
				"failureCount": failureCount,
				"retryAfter":   retryAfter.String(),
			},
		},
		FromState: Open,
		ToState:   Open,
	}
}

func NewCircuitBreakerHalfOpenError(successCount, requiredSuccesses int) *CircuitBreakerError {
	return &CircuitBreakerError{
		BaseError: &errors.BaseError{
			Message: fmt.Sprintf("circuit breaker is HALF-OPEN with active test in progress (successes: %d/%d required)", successCount, requiredSuccesses),
			Code:    "CIRCUIT_BREAKER_HALF_OPEN",
			Details: map[string]interface{}{
				"successCount":      successCount,
				"requiredSuccesses": requiredSuccesses,
			},
		},
		FromState: HalfOpen,
		ToState:   HalfOpen,
	}
}

// CircuitBreakerState represents the state of the circuit breaker.
type CircuitBreakerState int

// Possible circuit breaker states.
const (
	Closed CircuitBreakerState = iota
	Open
	HalfOpen
)

// String returns the string representation of the circuit breaker state.
func (s CircuitBreakerState) String() string {
	return [...]string{"Closed", "Open", "Half-Open"}[s]
}

// RuntimeCircuitBreaker implements a circuit breaker for container runtime operations.
type RuntimeCircuitBreaker struct {
	mu sync.RWMutex

	// State management
	state           CircuitBreakerState
	lastFailureTime time.Time
	lastStateChange time.Time

	// Configuration
	failureThreshold int           // Number of failures before opening
	successThreshold int           // Number of successes in half-open before closing
	recoveryTimeout  time.Duration // Time to wait before entering half-open
	halfOpenTimeout  time.Duration // Max time to stay in half-open before reopening

	// Counters
	consecutiveFailures  int
	consecutiveSuccesses int

	// Half-open specific
	halfOpenAttempts    int // Number of attempts made in half-open state
	maxHalfOpenAttempts int // Max concurrent attempts in half-open
	activeHalfOpenCalls int // Current number of active half-open calls

	OnStateChange func(from, to CircuitBreakerState)
	OnFailure     func(err error)

	logger *utils.ServiceLogger // Optional logger for debugging
}

// NewCircuitBreaker creates a new circuit breaker with sensible defaults
func NewCircuitBreaker() *RuntimeCircuitBreaker {

	logger := utils.NewServiceLogger("circuit_breaker")
	logger.Info("Initializing circuit breaker with default settings")

	return &RuntimeCircuitBreaker{
		state:               Closed,
		failureThreshold:    5,                // Open after 5 failures
		successThreshold:    2,                // Close after 2 successes in half-open
		recoveryTimeout:     30 * time.Second, // Wait 30s before trying half-open
		halfOpenTimeout:     10 * time.Second, // Stay in half-open for max 10s
		maxHalfOpenAttempts: 1,                // Only 1 concurrent test in half-open
		lastStateChange:     time.Now(),
		logger:              logger,
	}
}

func (cb *RuntimeCircuitBreaker) GetState() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

func (cb *RuntimeCircuitBreaker) Execute(operation func() error) error {
	cb.logger.Debug("Checking if operation is allowed by circuit breaker...")
	if err := cb.beforeExecution(); err != nil {
		return err
	}

	cb.logger.Debug("Operation allowed, executing...")

	err := operation()

	if err != nil {
		cb.logger.Debug("Operation failed, recording failure...")
	}

	cb.afterExecution(err)
	return err
}

// beforeExecution checks state and determines if operation can proceed
func (cb *RuntimeCircuitBreaker) beforeExecution() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()

	switch cb.state {
	case Open:
		// Check if it's time to try recovery
		if now.Sub(cb.lastStateChange) >= cb.recoveryTimeout {
			cb.transitionTo(HalfOpen, "Recovery timeout elapsed")
			// Allow this call to proceed as a test
			cb.activeHalfOpenCalls++
			cb.halfOpenAttempts++
			return nil
		}
		return NewCircuitBreakerOpenError(cb.consecutiveFailures, cb.recoveryTimeout-now.Sub(cb.lastStateChange))

	case HalfOpen:
		// Check if we've been in half-open too long without success
		if now.Sub(cb.lastStateChange) >= cb.halfOpenTimeout {
			cb.transitionTo(Open, "Half-open timeout exceeded without success")
			return NewCircuitBreakerStateChangeError(HalfOpen, Open, "circuit breaker returned to OPEN (half-open timeout)")
		}

		// Limit concurrent test calls in half-open state
		if cb.activeHalfOpenCalls >= cb.maxHalfOpenAttempts {
			return NewCircuitBreakerHalfOpenError(cb.consecutiveSuccesses, cb.successThreshold)
		}

		// Allow this call to proceed as a test
		cb.activeHalfOpenCalls++
		cb.halfOpenAttempts++
		return nil

	case Closed:
		// Normal operation
		return nil
	}

	return nil
}

// afterExecution records the result and updates state
func (cb *RuntimeCircuitBreaker) afterExecution(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == HalfOpen {
		cb.activeHalfOpenCalls--
	}

	if err != nil {
		cb.recordFailure(err)
	} else {
		cb.recordSuccess()
	}
}

// recordFailure handles a failed operation
func (cb *RuntimeCircuitBreaker) recordFailure(err error) {
	cb.consecutiveFailures++
	cb.consecutiveSuccesses = 0
	cb.lastFailureTime = time.Now()

	switch cb.state {
	case Closed:
		if cb.consecutiveFailures >= cb.failureThreshold {
			cb.transitionTo(Open, fmt.Sprintf("Failure threshold reached (%d failures)", cb.consecutiveFailures))
		}

	case HalfOpen:
		// Single failure in half-open immediately returns to open
		cb.transitionTo(Open, fmt.Sprintf("Failed during half-open test (attempt %d)", cb.halfOpenAttempts))
	default:
		// Already open, just log the failure
		if cb.OnFailure != nil {
			cb.OnFailure(err)
		}
	}
}

// recordSuccess handles a successful operation
func (cb *RuntimeCircuitBreaker) recordSuccess() {
	cb.consecutiveSuccesses++
	cb.consecutiveFailures = 0

	switch cb.state {
	case HalfOpen:
		if cb.consecutiveSuccesses >= cb.successThreshold {
			cb.transitionTo(Closed, fmt.Sprintf("Success threshold reached in half-open (%d successes)", cb.consecutiveSuccesses))
		}

	case Open:
		// This shouldn't happen, but handle it gracefully
		cb.transitionTo(HalfOpen, "Unexpected success in open state")

	case Closed:
		// Normal operation, nothing to do
	default:
		// Unknown state, reset to closed
		cb.transitionTo(Closed, "Unknown state, resetting to closed")
	}
}

// transitionTo changes the circuit breaker state
func (cb *RuntimeCircuitBreaker) transitionTo(newState CircuitBreakerState, reason string) {
	oldState := cb.state
	cb.state = newState
	cb.lastStateChange = time.Now()

	// Reset counters on state change
	if newState == HalfOpen {
		cb.halfOpenAttempts = 0
		cb.activeHalfOpenCalls = 0
		cb.consecutiveSuccesses = 0
	} else if newState == Closed {
		cb.consecutiveFailures = 0
		cb.consecutiveSuccesses = 0
	}

	fmt.Printf("[%s] Circuit breaker: %s -> %s (%s)\n",
		time.Now().Format("15:04:05"),
		oldState,
		newState,
		reason)

	cb.logger.Info(fmt.Sprintf("Circuit breaker state change: %s -> %s (%s)\n",
		oldState,
		newState,
		reason))
}

type DockerCircuitBreaker struct {
	*RuntimeCircuitBreaker
}
