package postgres

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/merraki/merraki-backend/internal/domain"
)

type PaymentRepository struct {
	db *sqlx.DB
}

func NewPaymentRepository(db *sqlx.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

func (r *PaymentRepository) Create(ctx context.Context, payment *domain.Payment) error {
	query := `
		INSERT INTO payments (
			order_id, gateway, gateway_order_id, gateway_payment_id, gateway_signature,
			amount_usd_cents, status,
			method, card_network, card_last4, bank, wallet, vpa,
			customer_email, customer_phone,
			signature_verified, gateway_response,
			error_code, error_description, error_source,
			gateway_fee_usd_cents, net_amount_usd_cents
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
			$21, $22
		) RETURNING id, created_at, updated_at
	`

	return r.db.QueryRowContext(
		ctx, query,
		payment.OrderID, payment.Gateway, payment.GatewayOrderID, payment.GatewayPaymentID, payment.GatewaySignature,
		payment.AmountUSDCents, payment.Status,
		payment.Method, payment.CardNetwork, payment.CardLast4, payment.Bank, payment.Wallet, payment.VPA,
		payment.CustomerEmail, payment.CustomerPhone,
		payment.SignatureVerified, payment.GatewayResponse,
		payment.ErrorCode, payment.ErrorDescription, payment.ErrorSource,
		payment.GatewayFeeUSDCents, payment.NetAmountUSDCents,
	).Scan(&payment.ID, &payment.CreatedAt, &payment.UpdatedAt)
}

func (r *PaymentRepository) FindByID(ctx context.Context, id int64) (*domain.Payment, error) {
	var payment domain.Payment
	err := r.db.GetContext(ctx, &payment, `SELECT * FROM payments WHERE id = $1`, id)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return &payment, err
}

func (r *PaymentRepository) FindByGatewayOrderID(ctx context.Context, gatewayOrderID string) (*domain.Payment, error) {
	var payment domain.Payment
	err := r.db.GetContext(ctx, &payment,
		`SELECT * FROM payments WHERE gateway_order_id = $1 ORDER BY created_at DESC LIMIT 1`, gatewayOrderID,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return &payment, err
}

func (r *PaymentRepository) FindByGatewayPaymentID(ctx context.Context, gatewayPaymentID string) (*domain.Payment, error) {
	var payment domain.Payment
	err := r.db.GetContext(ctx, &payment,
		`SELECT * FROM payments WHERE gateway_payment_id = $1 LIMIT 1`, gatewayPaymentID,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return &payment, err
}

func (r *PaymentRepository) GetByOrderID(ctx context.Context, orderID int64) ([]*domain.Payment, error) {
	var payments []*domain.Payment
	err := r.db.SelectContext(ctx, &payments,
		`SELECT * FROM payments WHERE order_id = $1 ORDER BY created_at DESC`, orderID,
	)
	return payments, err
}

func (r *PaymentRepository) Update(ctx context.Context, payment *domain.Payment) error {
	query := `
		UPDATE payments SET
			gateway_payment_id = $1, gateway_signature = $2,
			status = $3,
			method = $4, card_network = $5, card_last4 = $6, bank = $7, wallet = $8, vpa = $9,
			customer_email = $10, customer_phone = $11,
			signature_verified = $12, verified_at = $13, verification_attempts = $14,
			gateway_response = $15,
			error_code = $16, error_description = $17, error_source = $18,
			gateway_fee_usd_cents = $19, net_amount_usd_cents = $20,
			authorized_at = $21, captured_at = $22, failed_at = $23, refunded_at = $24,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $25
	`

	_, err := r.db.ExecContext(
		ctx, query,
		payment.GatewayPaymentID, payment.GatewaySignature,
		payment.Status,
		payment.Method, payment.CardNetwork, payment.CardLast4, payment.Bank, payment.Wallet, payment.VPA,
		payment.CustomerEmail, payment.CustomerPhone,
		payment.SignatureVerified, payment.VerifiedAt, payment.VerificationAttempts,
		payment.GatewayResponse,
		payment.ErrorCode, payment.ErrorDescription, payment.ErrorSource,
		payment.GatewayFeeUSDCents, payment.NetAmountUSDCents,
		payment.AuthorizedAt, payment.CapturedAt, payment.FailedAt, payment.RefundedAt,
		payment.ID,
	)
	return err
}

func (r *PaymentRepository) UpdateStatus(ctx context.Context, id int64, status domain.PaymentStatus) error {
	var timestampField string
	switch status {
	case domain.PaymentStatusAuthorized:
		timestampField = "authorized_at"
	case domain.PaymentStatusCaptured:
		timestampField = "captured_at"
	case domain.PaymentStatusFailed:
		timestampField = "failed_at"
	case domain.PaymentStatusRefunded:
		timestampField = "refunded_at"
	}

	query := `UPDATE payments SET status = $1, updated_at = CURRENT_TIMESTAMP`
	if timestampField != "" {
		query += ", " + timestampField + " = CURRENT_TIMESTAMP"
	}
	query += " WHERE id = $2"

	_, err := r.db.ExecContext(ctx, query, status, id)
	return err
}

func (r *PaymentRepository) MarkAsVerified(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE payments
		SET signature_verified = true, verified_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`, id)
	return err
}