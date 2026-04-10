package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/merraki/merraki-backend/internal/domain"
)

type OrderRepository struct {
	db *sqlx.DB
}

func NewOrderRepository(db *sqlx.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) Create(ctx context.Context, order *domain.Order) error {
	query := `
		INSERT INTO orders (
			order_number, customer_email, customer_name, customer_phone,
			customer_ip, customer_user_agent, customer_country,
			billing_name, billing_email, billing_phone,
			billing_address_line1, billing_address_line2,
			billing_city, billing_state, billing_country, billing_postal_code,
			subtotal_usd_cents, tax_amount_usd_cents, discount_amount_usd_cents, total_amount_usd_cents,
			payment_gateway, status, idempotency_key, metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
			$21, $22, $23, $24
		) RETURNING id, created_at, updated_at
	`

	return r.db.QueryRowContext(
		ctx, query,
		order.OrderNumber, order.CustomerEmail, order.CustomerName, order.CustomerPhone,
		order.CustomerIP, order.CustomerUserAgent, order.CustomerCountry,
		order.BillingName, order.BillingEmail, order.BillingPhone,
		order.BillingAddressLine1, order.BillingAddressLine2,
		order.BillingCity, order.BillingState, order.BillingCountry, order.BillingPostalCode,
		order.SubtotalUSDCents, order.TaxAmountUSDCents, order.DiscountAmountUSDCents, order.TotalAmountUSDCents,
		order.PaymentGateway, order.Status, order.IdempotencyKey, order.Metadata,
	).Scan(&order.ID, &order.CreatedAt, &order.UpdatedAt)
}

func (r *OrderRepository) FindByID(ctx context.Context, id int64) (*domain.Order, error) {
	var order domain.Order
	err := r.db.GetContext(ctx, &order, `SELECT * FROM orders WHERE id = $1`, id)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return &order, err
}

func (r *OrderRepository) FindByOrderNumber(ctx context.Context, orderNumber string) (*domain.Order, error) {
	var order domain.Order
	err := r.db.GetContext(ctx, &order, `SELECT * FROM orders WHERE order_number = $1`, orderNumber)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return &order, err
}

func (r *OrderRepository) FindByGatewayOrderID(ctx context.Context, gatewayOrderID string) (*domain.Order, error) {
	var order domain.Order
	err := r.db.GetContext(ctx, &order, `SELECT * FROM orders WHERE gateway_order_id = $1`, gatewayOrderID)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return &order, err
}

func (r *OrderRepository) FindByEmail(ctx context.Context, email string, limit, offset int) ([]*domain.Order, int, error) {
	var orders []*domain.Order
	var total int

	if err := r.db.GetContext(ctx, &total,
		`SELECT COUNT(*) FROM orders WHERE customer_email = $1`, email,
	); err != nil {
		return nil, 0, err
	}

	err := r.db.SelectContext(ctx, &orders, `
		SELECT * FROM orders
		WHERE customer_email = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, email, limit, offset)
	return orders, total, err
}

func (r *OrderRepository) GetAll(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*domain.Order, int, error) {
	var orders []*domain.Order
	var total int

	whereClauses := []string{"1=1"}
	args := []interface{}{}
	argPos := 1

	if status, ok := filters["status"].(domain.OrderStatus); ok {
		whereClauses = append(whereClauses, fmt.Sprintf("status = $%d", argPos))
		args = append(args, status)
		argPos++
	}

	if email, ok := filters["email"].(string); ok && email != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("customer_email ILIKE $%d", argPos))
		args = append(args, "%"+email+"%")
		argPos++
	}

	if orderNumber, ok := filters["order_number"].(string); ok && orderNumber != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("order_number ILIKE $%d", argPos))
		args = append(args, "%"+orderNumber+"%")
		argPos++
	}

	if startDate, ok := filters["start_date"].(string); ok && startDate != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("created_at >= $%d", argPos))
		args = append(args, startDate)
		argPos++
	}

	if endDate, ok := filters["end_date"].(string); ok && endDate != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("created_at <= $%d", argPos))
		args = append(args, endDate)
		argPos++
	}

	whereClause := strings.Join(whereClauses, " AND ")

	if err := r.db.GetContext(ctx, &total,
		fmt.Sprintf("SELECT COUNT(*) FROM orders WHERE %s", whereClause), args...,
	); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	query := fmt.Sprintf(`
		SELECT * FROM orders
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argPos, argPos+1)

	err := r.db.SelectContext(ctx, &orders, query, args...)
	return orders, total, err
}

func (r *OrderRepository) GetWithItems(ctx context.Context, id int64) (*domain.OrderWithItems, error) {
	order, err := r.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	var items []*domain.OrderItem
	if err = r.db.SelectContext(ctx, &items, `SELECT * FROM order_items WHERE order_id = $1 ORDER BY id`, id); err != nil {
		return nil, err
	}

	itemsSlice := make([]domain.OrderItem, len(items))
	for i, item := range items {
		itemsSlice[i] = *item
	}

	return &domain.OrderWithItems{Order: *order, Items: itemsSlice}, nil
}

func (r *OrderRepository) Update(ctx context.Context, order *domain.Order) error {
	query := `
		UPDATE orders SET
			customer_email = $1, customer_name = $2, customer_phone = $3,
			billing_name = $4, billing_email = $5, billing_phone = $6,
			billing_address_line1 = $7, billing_address_line2 = $8,
			billing_city = $9, billing_state = $10, billing_country = $11, billing_postal_code = $12,
			subtotal_usd_cents = $13, tax_amount_usd_cents = $14,
			discount_amount_usd_cents = $15, total_amount_usd_cents = $16,
			gateway_order_id = $17, gateway_payment_id = $18,
			status = $19,
			admin_reviewed_by = $20, admin_reviewed_at = $21,
			admin_notes = $22, rejection_reason = $23,
			downloads_enabled = $24, downloads_expires_at = $25,
			metadata = $26,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $27
	`

	_, err := r.db.ExecContext(
		ctx, query,
		order.CustomerEmail, order.CustomerName, order.CustomerPhone,
		order.BillingName, order.BillingEmail, order.BillingPhone,
		order.BillingAddressLine1, order.BillingAddressLine2,
		order.BillingCity, order.BillingState, order.BillingCountry, order.BillingPostalCode,
		order.SubtotalUSDCents, order.TaxAmountUSDCents,
		order.DiscountAmountUSDCents, order.TotalAmountUSDCents,
		order.GatewayOrderID, order.GatewayPaymentID,
		order.Status,
		order.AdminReviewedBy, order.AdminReviewedAt,
		order.AdminNotes, order.RejectionReason,
		order.DownloadsEnabled, order.DownloadsExpiresAt,
		order.Metadata,
		order.ID,
	)
	return err
}

func (r *OrderRepository) UpdateStatus(ctx context.Context, id int64, newStatus domain.OrderStatus, adminID *int64) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var order domain.Order
	if err = tx.GetContext(ctx, &order, "SELECT * FROM orders WHERE id = $1 FOR UPDATE", id); err != nil {
		return err
	}

	if !order.CanTransitionTo(newStatus) {
		return domain.ErrInvalidStateTransition
	}

	if _, err = tx.ExecContext(ctx,
		`UPDATE orders SET status = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`,
		newStatus, id,
	); err != nil {
		return err
	}

	triggeredBy := "system"
	if adminID != nil {
		triggeredBy = "admin"
	}
	if _, err = tx.ExecContext(ctx, `
		INSERT INTO order_state_transitions (order_id, from_status, to_status, triggered_by, admin_id)
		VALUES ($1, $2, $3, $4, $5)
	`, id, order.Status, newStatus, triggeredBy, adminID); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *OrderRepository) Approve(ctx context.Context, id int64, adminID int64, notes *string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now()
	expiresAt := now.AddDate(0, 0, 30)

	result, err := tx.ExecContext(ctx, `
		UPDATE orders SET
			status = $1,
			admin_reviewed_by = $2,
			admin_reviewed_at = $3,
			admin_notes = $4,
			downloads_enabled = true,
			downloads_expires_at = $5,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $6 AND status IN ('paid', 'admin_review')
	`, domain.OrderStatusApproved, adminID, now, notes, expiresAt, id)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.ErrInvalidStateTransition
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO order_state_transitions (order_id, to_status, triggered_by, admin_id, reason)
		VALUES ($1, $2, 'admin', $3, 'Order approved')
	`, id, domain.OrderStatusApproved, adminID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *OrderRepository) Reject(ctx context.Context, id int64, adminID int64, reason string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, `
		UPDATE orders SET
			status = $1,
			admin_reviewed_by = $2,
			admin_reviewed_at = CURRENT_TIMESTAMP,
			rejection_reason = $3,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $4 AND status IN ('paid', 'admin_review')
	`, domain.OrderStatusRejected, adminID, reason, id)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.ErrInvalidStateTransition
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO order_state_transitions (order_id, to_status, triggered_by, admin_id, reason)
		VALUES ($1, $2, 'admin', $3, $4)
	`, id, domain.OrderStatusRejected, adminID, reason)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// FIX 1: Added adminID int64 and gatewayOrderID string params to match interface
func (r *OrderRepository) MarkAsPaid(ctx context.Context, id int64, adminID int64, gatewayOrderID string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, `
		UPDATE orders
		SET status = $1, gateway_order_id = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3 AND status = 'pending'
	`, domain.OrderStatusPaid, gatewayOrderID, id)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.ErrInvalidStateTransition
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO order_state_transitions (order_id, from_status, to_status, triggered_by, admin_id)
		VALUES ($1, 'pending', 'paid', 'admin', $2)
	`, id, adminID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// FIX 2: Added adminID int64 param to match interface; cascade delete via SQL
func (r *OrderRepository) Delete(ctx context.Context, id int64, adminID int64) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	// Log the deletion in state transitions before deleting
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO order_state_transitions (order_id, to_status, triggered_by, admin_id, reason)
		VALUES ($1, 'cancelled', 'admin', $2, 'Order deleted by admin')
	`, id, adminID); err != nil {
		return err
	}

	for _, q := range []string{
		"DELETE FROM order_items WHERE order_id = $1",
		"DELETE FROM order_state_transitions WHERE order_id = $1",
		"DELETE FROM orders WHERE id = $1",
	} {
		if _, err := tx.ExecContext(ctx, q, id); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

// ============================================================================
// Analytics
// ============================================================================

// FIX 3: Changed return type from float64 to int64 to match interface
func (r *OrderRepository) GetRevenueByDateRange(ctx context.Context, startDate, endDate string) (float64, error) {
	var revenue int64
	err := r.db.GetContext(ctx, &revenue, `
		SELECT COALESCE(SUM(total_amount_usd_cents), 0)
		FROM orders
		WHERE status IN ('approved', 'paid')
		AND created_at >= $1 AND created_at <= $2
	`, startDate, endDate)
	// Convert cents to USD float at the boundary, matching the interface's float64 return
	return domain.CentsToUSD(revenue), err
}

func (r *OrderRepository) GetAverageOrderValue(ctx context.Context) (int64, error) {
	var avg int64
	err := r.db.GetContext(ctx, &avg, `
		SELECT COALESCE(AVG(total_amount_usd_cents), 0)
		FROM orders
		WHERE status IN ('approved', 'paid')
	`)
	return avg, err
}

func (r *OrderRepository) GetOrderCountByStatus(ctx context.Context) (map[domain.OrderStatus]int, error) {
	type StatusCount struct {
		Status domain.OrderStatus `db:"status"`
		Count  int                `db:"count"`
	}

	var counts []StatusCount
	if err := r.db.SelectContext(ctx, &counts, `SELECT status, COUNT(*) as count FROM orders GROUP BY status`); err != nil {
		return nil, err
	}

	result := make(map[domain.OrderStatus]int)
	for _, sc := range counts {
		result[sc.Status] = sc.Count
	}
	return result, nil
}