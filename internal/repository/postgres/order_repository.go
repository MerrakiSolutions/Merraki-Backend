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
			billing_city, billing_state, billing_country, billing_postal_code, subtotal, tax_amount, discount_amount, total_amount,
			payment_gateway, status, idempotency_key, metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
			$21, $22, $23, $24, $25
		) RETURNING id, created_at, updated_at
	`

	return r.db.QueryRowContext(
		ctx, query,
		order.OrderNumber, order.CustomerEmail, order.CustomerName, order.CustomerPhone,
		order.CustomerIP, order.CustomerUserAgent, order.CustomerCountry,
		order.BillingName, order.BillingEmail, order.BillingPhone,
		order.BillingAddressLine1, order.BillingAddressLine2,
		order.BillingCity, order.BillingState, order.BillingCountry, order.BillingPostalCode,
		order.Subtotal, order.TaxAmount, order.DiscountAmount, order.TotalAmount,
		order.PaymentGateway, order.Status, order.IdempotencyKey, order.Metadata,
	).Scan(&order.ID, &order.CreatedAt, &order.UpdatedAt)
}

func (r *OrderRepository) FindByID(ctx context.Context, id int64) (*domain.Order, error) {
	var order domain.Order
	query := `SELECT * FROM orders WHERE id = $1`

	err := r.db.GetContext(ctx, &order, query, id)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return &order, err
}

func (r *OrderRepository) FindByOrderNumber(ctx context.Context, orderNumber string) (*domain.Order, error) {
	var order domain.Order
	query := `SELECT * FROM orders WHERE order_number = $1`

	err := r.db.GetContext(ctx, &order, query, orderNumber)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return &order, err
}

func (r *OrderRepository) FindByGatewayOrderID(ctx context.Context, gatewayOrderID string) (*domain.Order, error) {
	var order domain.Order
	query := `SELECT * FROM orders WHERE gateway_order_id = $1`

	err := r.db.GetContext(ctx, &order, query, gatewayOrderID)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return &order, err
}

func (r *OrderRepository) FindByEmail(ctx context.Context, email string, limit, offset int) ([]*domain.Order, int, error) {
	var orders []*domain.Order
	var total int

	// Count total
	countQuery := `SELECT COUNT(*) FROM orders WHERE customer_email = $1`
	err := r.db.GetContext(ctx, &total, countQuery, email)
	if err != nil {
		return nil, 0, err
	}

	// Get data
	query := `
		SELECT * FROM orders 
		WHERE customer_email = $1 
		ORDER BY created_at DESC 
		LIMIT $2 OFFSET $3
	`
	err = r.db.SelectContext(ctx, &orders, query, email, limit, offset)
	return orders, total, err
}

func (r *OrderRepository) GetAll(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*domain.Order, int, error) {
	var orders []*domain.Order
	var total int

	// Build WHERE clause
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

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM orders WHERE %s", whereClause)
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	// Get data
	args = append(args, limit, offset)
	query := fmt.Sprintf(`
		SELECT * FROM orders 
		WHERE %s 
		ORDER BY created_at DESC 
		LIMIT $%d OFFSET $%d
	`, whereClause, argPos, argPos+1)

	err = r.db.SelectContext(ctx, &orders, query, args...)
	return orders, total, err
}

func (r *OrderRepository) GetWithItems(ctx context.Context, id int64) (*domain.OrderWithItems, error) {
	order, err := r.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	var items []*domain.OrderItem
	query := `SELECT * FROM order_items WHERE order_id = $1 ORDER BY id`
	err = r.db.SelectContext(ctx, &items, query, id)
	if err != nil {
		return nil, err
	}

	// Convert []*OrderItem to []OrderItem
	itemsSlice := make([]domain.OrderItem, len(items))
	for i, item := range items {
		itemsSlice[i] = *item
	}

	return &domain.OrderWithItems{
		Order: *order,
		Items: itemsSlice,
	}, nil
}

func (r *OrderRepository) Update(ctx context.Context, order *domain.Order) error {
	query := `
		UPDATE orders SET
			customer_email = $1, customer_name = $2, customer_phone = $3,
			billing_name = $4, billing_email = $5, billing_phone = $6,
			billing_address_line1 = $7, billing_address_line2 = $8,
			billing_city = $9, billing_state = $10, billing_country = $11, billing_postal_code = $12,
			currency = $13, subtotal = $14, tax_amount = $15, discount_amount = $16, total_amount = $17,
			gateway_order_id = $18, gateway_payment_id = $19, gateway_signature = $20,
			status = $21, previous_status = $22, status_updated_at = $23,
			admin_reviewed_by = $24, admin_reviewed_at = $25, admin_notes = $26, rejection_reason = $27,
			downloads_enabled = $28, downloads_expires_at = $29,
			paid_at = $30, approved_at = $31, rejected_at = $32, cancelled_at = $33, refunded_at = $34,
			metadata = $35
		WHERE id = $36
	`

	_, err := r.db.ExecContext(
		ctx, query,
		order.CustomerEmail, order.CustomerName, order.CustomerPhone,
		order.BillingName, order.BillingEmail, order.BillingPhone,
		order.BillingAddressLine1, order.BillingAddressLine2,
		order.BillingCity, order.BillingState, order.BillingCountry, order.BillingPostalCode, order.Subtotal, order.TaxAmount, order.DiscountAmount, order.TotalAmount,
		order.GatewayOrderID, order.GatewayPaymentID, order.GatewaySignature,
		order.Status, order.PreviousStatus, order.StatusUpdatedAt,
		order.AdminReviewedBy, order.AdminReviewedAt, order.AdminNotes, order.RejectionReason,
		order.DownloadsEnabled, order.DownloadsExpiresAt,
		order.PaidAt, order.ApprovedAt, order.RejectedAt, order.CancelledAt, order.RefundedAt,
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

	// Get current order
	var order domain.Order
	err = tx.GetContext(ctx, &order, "SELECT * FROM orders WHERE id = $1 FOR UPDATE", id)
	if err != nil {
		return err
	}

	// Validate transition
	if !order.CanTransitionTo(newStatus) {
		return domain.ErrInvalidStateTransition
	}

	// Update status
	now := time.Now()
	query := `
		UPDATE orders 
		SET status = $1, previous_status = $2, status_updated_at = $3
		WHERE id = $4
	`
	_, err = tx.ExecContext(ctx, query, newStatus, order.Status, now, id)
	if err != nil {
		return err
	}

	// Record transition
	transQuery := `
		INSERT INTO order_state_transitions (order_id, from_status, to_status, triggered_by, admin_id)
		VALUES ($1, $2, $3, $4, $5)
	`
	triggeredBy := "system"
	if adminID != nil {
		triggeredBy = "admin"
	}
	_, err = tx.ExecContext(ctx, transQuery, id, order.Status, newStatus, triggeredBy, adminID)
	if err != nil {
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
	expiresAt := now.AddDate(0, 0, 30) // 30 days download access

	query := `
		UPDATE orders SET
			status = $1,
			previous_status = status,
			status_updated_at = $2,
			admin_reviewed_by = $3,
			admin_reviewed_at = $4,
			admin_notes = $5,
			approved_at = $6,
			downloads_enabled = true,
			downloads_expires_at = $7
		WHERE id = $8 AND status IN ('paid', 'admin_review')
	`

	result, err := tx.ExecContext(ctx, query,
		domain.OrderStatusApproved, now, adminID, now, notes, now, expiresAt, id,
	)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.ErrInvalidStateTransition
	}

	// Record transition
	transQuery := `
		INSERT INTO order_state_transitions (order_id, to_status, triggered_by, admin_id, reason)
		VALUES ($1, $2, 'admin', $3, 'Order approved')
	`
	_, err = tx.ExecContext(ctx, transQuery, id, domain.OrderStatusApproved, adminID)
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

	now := time.Now()

	query := `
		UPDATE orders SET
			status = $1,
			previous_status = status,
			status_updated_at = $2,
			admin_reviewed_by = $3,
			admin_reviewed_at = $4,
			rejection_reason = $5,
			rejected_at = $6
		WHERE id = $7 AND status IN ('paid', 'admin_review')
	`

	result, err := tx.ExecContext(ctx, query,
		domain.OrderStatusRejected, now, adminID, now, reason, now, id,
	)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.ErrInvalidStateTransition
	}

	// Record transition
	transQuery := `
		INSERT INTO order_state_transitions (order_id, to_status, triggered_by, admin_id, reason)
		VALUES ($1, $2, 'admin', $3, $4)
	`
	_, err = tx.ExecContext(ctx, transQuery, id, domain.OrderStatusRejected, adminID, reason)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *OrderRepository) GetRevenueByDateRange(ctx context.Context, startDate, endDate string) (float64, error) {
	var revenue float64
	query := `
		SELECT COALESCE(SUM(total_amount), 0) 
		FROM orders 
		WHERE status IN ('approved', 'paid') 
		AND created_at >= $1 AND created_at <= $2
	`
	err := r.db.GetContext(ctx, &revenue, query, startDate, endDate)
	return revenue, err
}

func (r *OrderRepository) GetOrderCountByStatus(ctx context.Context) (map[domain.OrderStatus]int, error) {
	type StatusCount struct {
		Status domain.OrderStatus `db:"status"`
		Count  int                `db:"count"`
	}

	var counts []StatusCount
	query := `SELECT status, COUNT(*) as count FROM orders GROUP BY status`
	err := r.db.SelectContext(ctx, &counts, query)
	if err != nil {
		return nil, err
	}

	result := make(map[domain.OrderStatus]int)
	for _, sc := range counts {
		result[sc.Status] = sc.Count
	}

	return result, nil
}

func (r *OrderRepository) GetAverageOrderValue(ctx context.Context) (float64, error) {
	var avg float64
	query := `SELECT COALESCE(AVG(total_amount), 0) FROM orders WHERE status IN ('approved', 'paid')`
	err := r.db.GetContext(ctx, &avg, query)
	return avg, err
}

func (r *OrderRepository) MarkAsPaid(ctx context.Context, id int64, adminID int64, gatewayOrderID string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now()

	query := `
		UPDATE orders 
		SET status = $1, 
		    previous_status = status, 
		    status_updated_at = $2, 
		    paid_at = $3,
		    gateway_order_id = $4
		WHERE id = $5 AND status = 'pending'
	`

	result, err := tx.ExecContext(
		ctx,
		query,
		domain.OrderStatusPaid,
		now,
		now,
		gatewayOrderID,
		id,
	)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.ErrInvalidStateTransition
	}

	// Record transition
	transQuery := `
		INSERT INTO order_state_transitions (order_id, from_status, to_status, triggered_by, admin_id)
		VALUES ($1, 'pending', 'paid', 'admin', $2)
	`

	_, err = tx.ExecContext(ctx, transQuery, id, adminID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

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

	// delete children
	if _, err := tx.ExecContext(ctx,
		"DELETE FROM order_items WHERE order_id = $1", id); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx,
		"DELETE FROM order_state_transitions WHERE order_id = $1", id); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx,
		"DELETE FROM orders WHERE id = $1", id); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	committed = true
	return nil
}
