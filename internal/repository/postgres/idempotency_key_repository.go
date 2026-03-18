package postgres

import (
	"context"
	"database/sql"
	"github.com/jmoiron/sqlx"
	"github.com/merraki/merraki-backend/internal/domain"
)

type IdempotencyKeyRepository struct {
	db *sqlx.DB
}

func NewIdempotencyKeyRepository(db *sqlx.DB) *IdempotencyKeyRepository {
	return &IdempotencyKeyRepository{db: db}
}

func (r *IdempotencyKeyRepository) Create(ctx context.Context, key *domain.IdempotencyKey) error {
	query := `
		INSERT INTO idempotency_keys (
			key, operation_type, entity_type, entity_id,
			http_status_code, response_body, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`

	return r.db.QueryRowContext(
		ctx, query,
		key.Key, key.OperationType, key.EntityType, key.EntityID,
		key.HTTPStatusCode, key.ResponseBody, key.ExpiresAt,
	).Scan(&key.ID, &key.CreatedAt)
}

func (r *IdempotencyKeyRepository) FindByKey(ctx context.Context, key string) (*domain.IdempotencyKey, error) {
	var ik domain.IdempotencyKey
	query := `SELECT * FROM idempotency_keys WHERE key = $1 AND expires_at > CURRENT_TIMESTAMP`

	err := r.db.GetContext(ctx, &ik, query, key)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return &ik, err
}

func (r *IdempotencyKeyRepository) CleanupExpired(ctx context.Context) (int64, error) {
	query := `DELETE FROM idempotency_keys WHERE expires_at < CURRENT_TIMESTAMP`
	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}