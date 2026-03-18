package postgres

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/merraki/merraki-backend/internal/domain"
)

type OrderItemRepository struct {
	db *sqlx.DB
}

func NewOrderItemRepository(db *sqlx.DB) *OrderItemRepository {
	return &OrderItemRepository{db: db}
}

func (r *OrderItemRepository) Create(ctx context.Context, item *domain.OrderItem) error {
	query := `
		INSERT INTO order_items (
			order_id, template_id, template_name, template_slug, template_version,
			unit_price, quantity, subtotal,
			file_url, file_format, file_size_mb
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at
	`

	return r.db.QueryRowContext(
		ctx, query,
		item.OrderID, item.TemplateID, item.TemplateName, item.TemplateSlug, item.TemplateVersion,
		item.UnitPrice, item.Quantity, item.Subtotal,
		item.FileURL, item.FileFormat, item.FileSizeMB,
	).Scan(&item.ID, &item.CreatedAt)
}

func (r *OrderItemRepository) CreateBatch(ctx context.Context, items []*domain.OrderItem) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO order_items (
			order_id, template_id, template_name, template_slug, template_version,
			unit_price, quantity, subtotal,
			file_url, file_format, file_size_mb
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at
	`

	for _, item := range items {
		err := tx.QueryRowContext(
			ctx, query,
			item.OrderID, item.TemplateID, item.TemplateName, item.TemplateSlug, item.TemplateVersion,
			item.UnitPrice, item.Quantity, item.Subtotal,
			item.FileURL, item.FileFormat, item.FileSizeMB,
		).Scan(&item.ID, &item.CreatedAt)

		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *OrderItemRepository) GetByOrderID(ctx context.Context, orderID int64) ([]*domain.OrderItem, error) {
	var items []*domain.OrderItem
	query := `SELECT * FROM order_items WHERE order_id = $1 ORDER BY id`
	err := r.db.SelectContext(ctx, &items, query, orderID)
	return items, err
}

func (r *OrderItemRepository) GetByID(ctx context.Context, id int64) (*domain.OrderItem, error) {
	var item domain.OrderItem
	query := `SELECT * FROM order_items WHERE id = $1`
	err := r.db.GetContext(ctx, &item, query, id)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *OrderItemRepository) IncrementDownloadCount(ctx context.Context, id int64) error {
	query := `
		UPDATE order_items 
		SET download_count = download_count + 1, last_downloaded_at = CURRENT_TIMESTAMP 
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}