package postgres

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/merraki/merraki-backend/internal/domain"
)

type CircuitBreakerRepository struct {
	db *sqlx.DB
}

func NewCircuitBreakerRepository(db *sqlx.DB) *CircuitBreakerRepository {
	return &CircuitBreakerRepository{db: db}
}

func (r *CircuitBreakerRepository) GetByServiceName(ctx context.Context, serviceName string) (*domain.CircuitBreakerState, error) {
	var state domain.CircuitBreakerState
	query := `SELECT * FROM circuit_breaker_state WHERE service_name = $1`

	err := r.db.GetContext(ctx, &state, query, serviceName)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return &state, err
}

func (r *CircuitBreakerRepository) UpdateState(ctx context.Context, state *domain.CircuitBreakerState) error {
	query := `
		UPDATE circuit_breaker_state SET
			state = $1, failure_count = $2, success_count = $3,
			last_failure_at = $4, last_success_at = $5,
			state_changed_at = $6, next_attempt_at = $7
		WHERE service_name = $8
	`

	_, err := r.db.ExecContext(
		ctx, query,
		state.State, state.FailureCount, state.SuccessCount,
		state.LastFailureAt, state.LastSuccessAt,
		state.StateChangedAt, state.NextAttemptAt,
		state.ServiceName,
	)
	return err
}

func (r *CircuitBreakerRepository) IncrementFailure(ctx context.Context, serviceName string) error {
	query := `
		UPDATE circuit_breaker_state 
		SET failure_count = failure_count + 1, last_failure_at = CURRENT_TIMESTAMP
		WHERE service_name = $1
	`
	_, err := r.db.ExecContext(ctx, query, serviceName)
	return err
}

func (r *CircuitBreakerRepository) IncrementSuccess(ctx context.Context, serviceName string) error {
	query := `
		UPDATE circuit_breaker_state 
		SET success_count = success_count + 1, last_success_at = CURRENT_TIMESTAMP
		WHERE service_name = $1
	`
	_, err := r.db.ExecContext(ctx, query, serviceName)
	return err
}

func (r *CircuitBreakerRepository) ResetCounts(ctx context.Context, serviceName string) error {
	query := `
		UPDATE circuit_breaker_state 
		SET failure_count = 0, success_count = 0
		WHERE service_name = $1
	`
	_, err := r.db.ExecContext(ctx, query, serviceName)
	return err
}