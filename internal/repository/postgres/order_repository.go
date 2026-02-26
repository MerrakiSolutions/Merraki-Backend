package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/merraki/merraki-backend/internal/domain"
)

type OrderRepository struct {
	db *Database
}

func NewOrderRepository(db *Database) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) Create(ctx context.Context, order *domain.Order) error {
	query := `
		INSERT INTO orders 
		(order_number, customer_email, customer_name, customer_phone, 
		 subtotal_inr, discount_inr, tax_inr, total_inr, currency_code, 
		 exchange_rate, total_local, payment_method, status, payment_status,
		 download_token, max_downloads, download_expires_at, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
		RETURNING id, created_at, updated_at`

	return r.db.DB.QueryRowContext(
		ctx, query,
		order.OrderNumber, order.CustomerEmail, order.CustomerName, order.CustomerPhone,
		order.SubtotalINR, order.DiscountINR, order.TaxINR, order.TotalINR,
		order.CurrencyCode, order.ExchangeRate, order.TotalLocal, order.PaymentMethod,
		order.Status, order.PaymentStatus, order.DownloadToken, order.MaxDownloads,
		order.DownloadExpiresAt, order.IPAddress, order.UserAgent,
	).Scan(&order.ID, &order.CreatedAt, &order.UpdatedAt)
}

func (r *OrderRepository) FindByID(ctx context.Context, id int64) (*domain.Order, error) {
	var order domain.Order
	query := `SELECT * FROM orders WHERE id = $1`

	err := r.db.DB.GetContext(ctx, &order, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &order, err
}

func (r *OrderRepository) FindByOrderNumber(ctx context.Context, orderNumber string) (*domain.Order, error) {
	var order domain.Order
	query := `SELECT * FROM orders WHERE order_number = $1`

	err := r.db.DB.GetContext(ctx, &order, query, orderNumber)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &order, err
}

func (r *OrderRepository) FindByRazorpayOrderID(ctx context.Context, razorpayOrderID string) (*domain.Order, error) {
	var order domain.Order
	query := `SELECT * FROM orders WHERE razorpay_order_id = $1`

	err := r.db.DB.GetContext(ctx, &order, query, razorpayOrderID)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &order, err
}

func (r *OrderRepository) FindByOrderNumberAndEmail(ctx context.Context, orderNumber, email string) (*domain.Order, error) {
	var order domain.Order
	query := `SELECT * FROM orders WHERE order_number = $1 AND customer_email = $2`

	err := r.db.DB.GetContext(ctx, &order, query, orderNumber, email)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &order, err
}

func (r *OrderRepository) FindByDownloadToken(ctx context.Context, token string) (*domain.Order, error) {
	var order domain.Order
	query := `SELECT * FROM orders WHERE download_token = $1`

	err := r.db.DB.GetContext(ctx, &order, query, token)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &order, err
}

func (r *OrderRepository) Update(ctx context.Context, order *domain.Order) error {
	query := `
		UPDATE orders 
		SET status = $1, payment_status = $2, razorpay_order_id = $3, 
		    razorpay_payment_id = $4, razorpay_signature = $5, 
		    approved_by = $6, approved_at = $7, rejection_reason = $8,
		    download_count = $9, paid_at = $10, completed_at = $11, updated_at = NOW()
		WHERE id = $12
		RETURNING updated_at`

	return r.db.DB.QueryRowContext(
		ctx, query,
		order.Status, order.PaymentStatus, order.RazorpayOrderID,
		order.RazorpayPaymentID, order.RazorpaySignature,
		order.ApprovedBy, order.ApprovedAt, order.RejectionReason,
		order.DownloadCount, order.PaidAt, order.CompletedAt, order.ID,
	).Scan(&order.UpdatedAt)
}

func (r *OrderRepository) UpdatePaymentSuccess(ctx context.Context, id int64, razorpayPaymentID, razorpaySignature string) error {
	now := time.Now()
	query := `
		UPDATE orders 
		SET payment_status = 'success', 
		    status = 'paid',
		    razorpay_payment_id = $1,
		    razorpay_signature = $2,
		    paid_at = $3,
		    updated_at = NOW()
		WHERE id = $4`

	_, err := r.db.DB.ExecContext(ctx, query, razorpayPaymentID, razorpaySignature, now, id)
	return err
}

func (r *OrderRepository) UpdatePaymentFailed(ctx context.Context, id int64) error {
	query := `
		UPDATE orders 
		SET payment_status = 'failed', 
		    status = 'failed',
		    updated_at = NOW()
		WHERE id = $1`

	_, err := r.db.DB.ExecContext(ctx, query, id)
	return err
}

func (r *OrderRepository) ApproveOrder(ctx context.Context, id, adminID int64) error {
	now := time.Now()
	query := `
		UPDATE orders 
		SET status = 'approved',
		    approved_by = $1,
		    approved_at = $2,
		    updated_at = NOW()
		WHERE id = $3`

	_, err := r.db.DB.ExecContext(ctx, query, adminID, now, id)
	return err
}

func (r *OrderRepository) RejectOrder(ctx context.Context, id, adminID int64, reason string) error {
	now := time.Now()
	query := `
		UPDATE orders 
		SET status = 'rejected',
		    rejection_reason = $1,
		    approved_by = $2,
		    approved_at = $3,
		    updated_at = NOW()
		WHERE id = $4`

	_, err := r.db.DB.ExecContext(ctx, query, reason, adminID, now, id)
	return err
}

func (r *OrderRepository) IncrementDownloadCount(ctx context.Context, id int64) error {
	query := `UPDATE orders SET download_count = download_count + 1, updated_at = NOW() WHERE id = $1`
	_, err := r.db.DB.ExecContext(ctx, query, id)
	return err
}

func (r *OrderRepository) MarkCompleted(ctx context.Context, id int64) error {
	now := time.Now()
	query := `
		UPDATE orders 
		SET status = 'completed', completed_at = $1, updated_at = NOW()
		WHERE id = $2`

	_, err := r.db.DB.ExecContext(ctx, query, now, id)
	return err
}

func (r *OrderRepository) GetAll(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*domain.Order, int, error) {
	var orders []*domain.Order
	
	query := `SELECT * FROM orders WHERE 1=1`
	countQuery := `SELECT COUNT(*) FROM orders WHERE 1=1`
	args := []interface{}{}
	argCount := 1

	if status, ok := filters["status"].(string); ok && status != "" {
		query += fmt.Sprintf(" AND status = $%d", argCount)
		countQuery += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, status)
		argCount++
	}

	if paymentStatus, ok := filters["payment_status"].(string); ok && paymentStatus != "" {
		query += fmt.Sprintf(" AND payment_status = $%d", argCount)
		countQuery += fmt.Sprintf(" AND payment_status = $%d", argCount)
		args = append(args, paymentStatus)
		argCount++
	}

	if search, ok := filters["search"].(string); ok && search != "" {
		query += fmt.Sprintf(" AND (customer_email ILIKE $%d OR order_number ILIKE $%d)", argCount, argCount)
		countQuery += fmt.Sprintf(" AND (customer_email ILIKE $%d OR order_number ILIKE $%d)", argCount, argCount)
		args = append(args, "%"+search+"%")
		argCount++
	}

	if startDate, ok := filters["start_date"].(string); ok && startDate != "" {
		query += fmt.Sprintf(" AND created_at >= $%d", argCount)
		countQuery += fmt.Sprintf(" AND created_at >= $%d", argCount)
		args = append(args, startDate)
		argCount++
	}

	if endDate, ok := filters["end_date"].(string); ok && endDate != "" {
		query += fmt.Sprintf(" AND created_at <= $%d", argCount)
		countQuery += fmt.Sprintf(" AND created_at <= $%d", argCount)
		args = append(args, endDate)
		argCount++
	}

	sort := "created_at DESC"
	if sortParam, ok := filters["sort"].(string); ok {
		switch sortParam {
		case "newest":
			sort = "created_at DESC"
		case "oldest":
			sort = "created_at ASC"
		case "amount_high":
			sort = "total_inr DESC"
		case "amount_low":
			sort = "total_inr ASC"
		}
	}

	query += " ORDER BY " + sort
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argCount, argCount+1)

	var total int
	if err := r.db.DB.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	if err := r.db.DB.SelectContext(ctx, &orders, query, args...); err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

func (r *OrderRepository) GetPendingApprovals(ctx context.Context, limit, offset int) ([]*domain.Order, int, error) {
	var orders []*domain.Order
	
	query := `
		SELECT * FROM orders 
		WHERE status = 'paid' 
		ORDER BY paid_at ASC 
		LIMIT $1 OFFSET $2`

	countQuery := `SELECT COUNT(*) FROM orders WHERE status = 'paid'`

	var total int
	if err := r.db.DB.GetContext(ctx, &total, countQuery); err != nil {
		return nil, 0, err
	}

	if err := r.db.DB.SelectContext(ctx, &orders, query, limit, offset); err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

// Order Items
func (r *OrderRepository) CreateOrderItem(ctx context.Context, item *domain.OrderItem) error {
	query := `
		INSERT INTO order_items 
		(order_id, template_id, template_title, template_slug, template_file_url, price_inr)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`

	return r.db.DB.QueryRowContext(
		ctx, query,
		item.OrderID, item.TemplateID, item.TemplateTitle,
		item.TemplateSlug, item.TemplateFileURL, item.PriceINR,
	).Scan(&item.ID, &item.CreatedAt)
}

func (r *OrderRepository) GetOrderItems(ctx context.Context, orderID int64) ([]*domain.OrderItem, error) {
	var items []*domain.OrderItem
	query := `SELECT * FROM order_items WHERE order_id = $1`

	err := r.db.DB.SelectContext(ctx, &items, query, orderID)
	return items, err
}

// Order Status History
func (r *OrderRepository) GetStatusHistory(ctx context.Context, orderID int64) ([]*domain.OrderStatusHistory, error) {
	var history []*domain.OrderStatusHistory
	query := `
		SELECT * FROM order_status_history 
		WHERE order_id = $1 
		ORDER BY created_at ASC`

	err := r.db.DB.SelectContext(ctx, &history, query, orderID)
	return history, err
}

// Download Logs
func (r *OrderRepository) LogDownload(ctx context.Context, log *domain.DownloadLog) error {
	query := `
		INSERT INTO download_logs 
		(order_id, template_id, template_title, download_type, file_size_bytes, 
		 ip_address, user_agent, status, error_message)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, downloaded_at`

	return r.db.DB.QueryRowContext(
		ctx, query,
		log.OrderID, log.TemplateID, log.TemplateTitle, log.DownloadType,
		log.FileSizeBytes, log.IPAddress, log.UserAgent, log.Status, log.ErrorMessage,
	).Scan(&log.ID, &log.DownloadedAt)
}

func (r *OrderRepository) GetDownloadLogs(ctx context.Context, orderID int64) ([]*domain.DownloadLog, error) {
	var logs []*domain.DownloadLog
	query := `
		SELECT * FROM download_logs 
		WHERE order_id = $1 
		ORDER BY downloaded_at DESC`

	err := r.db.DB.SelectContext(ctx, &logs, query, orderID)
	return logs, err
}

// Analytics
func (r *OrderRepository) GetRevenueAnalytics(ctx context.Context, startDate, endDate string, groupBy string) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	
	dateFormat := "DATE(created_at)"
	if groupBy == "month" {
		dateFormat = "DATE_TRUNC('month', created_at)"
	} else if groupBy == "week" {
		dateFormat = "DATE_TRUNC('week', created_at)"
	}

	query := fmt.Sprintf(`
		SELECT 
			%s as period,
			COUNT(*) as orders_count,
			SUM(total_inr) as revenue_inr,
			AVG(total_inr) as avg_order_value_inr
		FROM orders
		WHERE status = 'completed'
		  AND created_at >= $1
		  AND created_at <= $2
		GROUP BY period
		ORDER BY period ASC`, dateFormat)

	rows, err := r.db.DB.QueryContext(ctx, query, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var period time.Time
		var ordersCount int
		var revenueINR, avgOrderValue int64

		if err := rows.Scan(&period, &ordersCount, &revenueINR, &avgOrderValue); err != nil {
			return nil, err
		}

		results = append(results, map[string]interface{}{
			"period":              period,
			"orders_count":        ordersCount,
			"revenue_inr":         revenueINR,
			"avg_order_value_inr": avgOrderValue,
		})
	}

	return results, nil
}