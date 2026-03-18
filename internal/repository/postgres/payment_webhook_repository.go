package postgres

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/merraki/merraki-backend/internal/domain"
)

type PaymentWebhookRepository struct {
	db *sqlx.DB
}

func NewPaymentWebhookRepository(db *sqlx.DB) *PaymentWebhookRepository {
	return &PaymentWebhookRepository{db: db}
}

func (r *PaymentWebhookRepository) Create(ctx context.Context, webhook *domain.PaymentWebhook) error {
	query := `
		INSERT INTO payment_webhooks (
			webhook_id, event_type, order_id, payment_id,
			gateway_order_id, gateway_payment_id,
			payload, signature, signature_verified,
			source_ip, user_agent, max_retries
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at
	`

	return r.db.QueryRowContext(
		ctx, query,
		webhook.WebhookID, webhook.EventType, webhook.OrderID, webhook.PaymentID,
		webhook.GatewayOrderID, webhook.GatewayPaymentID,
		webhook.Payload, webhook.Signature, webhook.SignatureVerified,
		webhook.SourceIP, webhook.UserAgent, webhook.MaxRetries,
	).Scan(&webhook.ID, &webhook.CreatedAt)
}

func (r *PaymentWebhookRepository) FindByID(ctx context.Context, id int64) (*domain.PaymentWebhook, error) {
	var webhook domain.PaymentWebhook
	query := `SELECT * FROM payment_webhooks WHERE id = $1`

	err := r.db.GetContext(ctx, &webhook, query, id)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return &webhook, err
}

func (r *PaymentWebhookRepository) GetUnprocessed(ctx context.Context, limit int) ([]*domain.PaymentWebhook, error) {
	var webhooks []*domain.PaymentWebhook
	query := `
		SELECT * FROM payment_webhooks 
		WHERE processed = false AND retry_count < max_retries
		ORDER BY created_at ASC 
		LIMIT $1
	`
	err := r.db.SelectContext(ctx, &webhooks, query, limit)
	return webhooks, err
}

func (r *PaymentWebhookRepository) MarkAsProcessed(ctx context.Context, id int64) error {
	query := `
		UPDATE payment_webhooks 
		SET processed = true, processed_at = CURRENT_TIMESTAMP 
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *PaymentWebhookRepository) IncrementRetryCount(ctx context.Context, id int64, errorMsg string) error {
	query := `
		UPDATE payment_webhooks 
		SET retry_count = retry_count + 1, processing_error = $1 
		WHERE id = $2
	`
	_, err := r.db.ExecContext(ctx, query, errorMsg, id)
	return err
}