package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/merraki/merraki-backend/internal/domain"
)

type DownloadTokenRepository struct {
	db *sqlx.DB
}

func NewDownloadTokenRepository(db *sqlx.DB) *DownloadTokenRepository {
	return &DownloadTokenRepository{db: db}
}

func (r *DownloadTokenRepository) Create(ctx context.Context, token *domain.DownloadToken) error {
	query := `
		INSERT INTO download_tokens (
			token, order_id, order_item_id, template_id,
			customer_email, expires_at, max_downloads, created_ip
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at
	`

	return r.db.QueryRowContext(
		ctx, query,
		token.Token, token.OrderID, token.OrderItemID, token.TemplateID,
		token.CustomerEmail, token.ExpiresAt, token.MaxDownloads, token.CreatedIP,
	).Scan(&token.ID, &token.CreatedAt)
}

func (r *DownloadTokenRepository) FindByToken(ctx context.Context, token string) (*domain.DownloadToken, error) {
	var dt domain.DownloadToken
	query := `SELECT * FROM download_tokens WHERE token = $1`

	err := r.db.GetContext(ctx, &dt, query, token)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return &dt, err
}

func (r *DownloadTokenRepository) GetByOrderID(ctx context.Context, orderID int64) ([]*domain.DownloadToken, error) {
	var tokens []*domain.DownloadToken
	query := `SELECT * FROM download_tokens WHERE order_id = $1 ORDER BY created_at DESC`
	err := r.db.SelectContext(ctx, &tokens, query, orderID)
	return tokens, err
}

func (r *DownloadTokenRepository) GetByEmail(ctx context.Context, email string) ([]*domain.DownloadToken, error) {
	var tokens []*domain.DownloadToken
	query := `
		SELECT * FROM download_tokens 
		WHERE customer_email = $1 
		AND is_revoked = false 
		AND expires_at > CURRENT_TIMESTAMP
		ORDER BY created_at DESC
	`
	err := r.db.SelectContext(ctx, &tokens, query, email)
	return tokens, err
}

func (r *DownloadTokenRepository) IncrementDownloadCount(ctx context.Context, id int64) error {
	query := `
		UPDATE download_tokens 
		SET download_count = download_count + 1, last_used_at = CURRENT_TIMESTAMP 
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *DownloadTokenRepository) Revoke(ctx context.Context, id int64, adminID int64, reason string) error {
	query := `
		UPDATE download_tokens 
		SET is_revoked = true, revoked_at = CURRENT_TIMESTAMP, revoked_by = $1, revoked_reason = $2 
		WHERE id = $3
	`
	_, err := r.db.ExecContext(ctx, query, adminID, reason, id)
	return err
}

func (r *DownloadTokenRepository) CleanupExpired(ctx context.Context) (int64, error) {
	// Optionally delete expired tokens (or just leave them for audit)
	query := `DELETE FROM download_tokens WHERE expires_at < CURRENT_TIMESTAMP AND created_at < $1`
	
	// Delete tokens older than 90 days past expiration
	cutoff := time.Now().AddDate(0, 0, -90)
	result, err := r.db.ExecContext(ctx, query, cutoff)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}