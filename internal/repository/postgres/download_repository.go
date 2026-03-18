package postgres

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/merraki/merraki-backend/internal/domain"
)

type DownloadRepository struct {
	db *sqlx.DB
}

func NewDownloadRepository(db *sqlx.DB) *DownloadRepository {
	return &DownloadRepository{db: db}
}

func (r *DownloadRepository) Create(ctx context.Context, download *domain.Download) error {
	query := `
		INSERT INTO downloads (
			token_id, order_id, order_item_id, template_id,
			customer_email, ip_address, user_agent, country,
			file_url, file_size_bytes
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, started_at, created_at
	`

	return r.db.QueryRowContext(
		ctx, query,
		download.TokenID, download.OrderID, download.OrderItemID, download.TemplateID,
		download.CustomerEmail, download.IPAddress, download.UserAgent, download.Country,
		download.FileURL, download.FileSizeBytes,
	).Scan(&download.ID, &download.StartedAt, &download.CreatedAt)
}

func (r *DownloadRepository) GetByTokenID(ctx context.Context, tokenID int64) ([]*domain.Download, error) {
	var downloads []*domain.Download
	query := `SELECT * FROM downloads WHERE token_id = $1 ORDER BY started_at DESC`
	err := r.db.SelectContext(ctx, &downloads, query, tokenID)
	return downloads, err
}

func (r *DownloadRepository) GetByOrderID(ctx context.Context, orderID int64) ([]*domain.Download, error) {
	var downloads []*domain.Download
	query := `SELECT * FROM downloads WHERE order_id = $1 ORDER BY started_at DESC`
	err := r.db.SelectContext(ctx, &downloads, query, orderID)
	return downloads, err
}

func (r *DownloadRepository) MarkAsCompleted(ctx context.Context, id int64, durationMS int) error {
	query := `
		UPDATE downloads 
		SET completed_at = CURRENT_TIMESTAMP, download_duration_ms = $1, failed = false 
		WHERE id = $2
	`
	_, err := r.db.ExecContext(ctx, query, durationMS, id)
	return err
}

func (r *DownloadRepository) MarkAsFailed(ctx context.Context, id int64, errorMsg string) error {
	query := `
		UPDATE downloads 
		SET failed = true, error_message = $1 
		WHERE id = $2
	`
	_, err := r.db.ExecContext(ctx, query, errorMsg, id)
	return err
}