package service

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/merraki/merraki-backend/internal/domain"
	"github.com/merraki/merraki-backend/internal/pkg/logger"
	"github.com/merraki/merraki-backend/internal/repository"
	"go.uber.org/zap"
)

// ============================================================================
// CIRCUIT BREAKER - Protect external services
// ============================================================================

var (
	ErrCircuitOpen     = errors.New("circuit breaker is open")
	ErrServiceUnavailable = errors.New("service temporarily unavailable")
)

type CircuitBreaker struct {
	serviceName string
	repo        repository.CircuitBreakerRepository
	mu          sync.RWMutex
	
	// In-memory cache to avoid DB hits on every request
	state            string
	failureCount     int
	successCount     int
	lastStateChange  time.Time
	nextAttemptAt    time.Time
}

func NewCircuitBreaker(serviceName string, repo repository.CircuitBreakerRepository) *CircuitBreaker {
	cb := &CircuitBreaker{
		serviceName: serviceName,
		repo:        repo,
		state:       "closed",
	}

	// Load initial state from DB
	cb.loadState(context.Background())

	return cb
}

func (cb *CircuitBreaker) loadState(ctx context.Context) {
	state, err := cb.repo.GetByServiceName(ctx, cb.serviceName)
	if err != nil {
		logger.Warn("Failed to load circuit breaker state, using defaults",
			zap.String("service", cb.serviceName),
			zap.Error(err),
		)
		return
	}

	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = state.State
	cb.failureCount = state.FailureCount
	cb.successCount = state.SuccessCount
	cb.lastStateChange = state.StateChangedAt
	if state.NextAttemptAt != nil {
		cb.nextAttemptAt = *state.NextAttemptAt
	}
}

func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() (interface{}, error)) (interface{}, error) {
	// Check if circuit is open
	if !cb.canExecute() {
		logger.Warn("Circuit breaker is open",
			zap.String("service", cb.serviceName),
		)
		return nil, ErrCircuitOpen
	}

	// Execute function
	result, err := fn()

	// Record result
	if err != nil {
		cb.recordFailure(ctx)
		return nil, err
	}

	cb.recordSuccess(ctx)
	return result, nil
}

func (cb *CircuitBreaker) canExecute() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case "closed":
		return true
	case "open":
		// Check if enough time has passed to try again
		return time.Now().After(cb.nextAttemptAt)
	case "half_open":
		return true
	default:
		return false
	}
}

func (cb *CircuitBreaker) recordSuccess(ctx context.Context) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.successCount++

	// Update DB
	_ = cb.repo.IncrementSuccess(ctx, cb.serviceName)

	// State transitions
	if cb.state == "half_open" && cb.successCount >= 2 {
		// Transition to closed
		cb.state = "closed"
		cb.failureCount = 0
		cb.successCount = 0
		cb.lastStateChange = time.Now()

		logger.Info("Circuit breaker closed",
			zap.String("service", cb.serviceName),
		)

		// Update DB
		state := &domain.CircuitBreakerState{
			ServiceName:    cb.serviceName,
			State:          "closed",
			FailureCount:   0,
			SuccessCount:   0,
			StateChangedAt: cb.lastStateChange,
		}
		_ = cb.repo.UpdateState(ctx, state)
	}
}

func (cb *CircuitBreaker) recordFailure(ctx context.Context) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++

	// Update DB
	_ = cb.repo.IncrementFailure(ctx, cb.serviceName)

	// State transitions
	if cb.state == "closed" && cb.failureCount >= 5 {
		// Transition to open
		cb.state = "open"
		cb.lastStateChange = time.Now()
		cb.nextAttemptAt = time.Now().Add(60 * time.Second)

		logger.Error("Circuit breaker opened",
			zap.String("service", cb.serviceName),
			zap.Int("failure_count", cb.failureCount),
		)

		// Update DB
		state := &domain.CircuitBreakerState{
			ServiceName:    cb.serviceName,
			State:          "open",
			FailureCount:   cb.failureCount,
			SuccessCount:   0,
			StateChangedAt: cb.lastStateChange,
			NextAttemptAt:  &cb.nextAttemptAt,
		}
		_ = cb.repo.UpdateState(ctx, state)
	} else if cb.state == "half_open" {
		// Transition back to open
		cb.state = "open"
		cb.nextAttemptAt = time.Now().Add(60 * time.Second)

		logger.Warn("Circuit breaker reopened",
			zap.String("service", cb.serviceName),
		)

		// Update DB
		state := &domain.CircuitBreakerState{
			ServiceName:    cb.serviceName,
			State:          "open",
			StateChangedAt: time.Now(),
			NextAttemptAt:  &cb.nextAttemptAt,
		}
		_ = cb.repo.UpdateState(ctx, state)
	}
}