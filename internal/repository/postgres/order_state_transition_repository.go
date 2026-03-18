package postgres

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/merraki/merraki-backend/internal/domain"
)

type OrderStateTransitionRepository struct {
	db *sqlx.DB
}

func NewOrderStateTransitionRepository(db *sqlx.DB) *OrderStateTransitionRepository {
	return &OrderStateTransitionRepository{db: db}
}

func (r *OrderStateTransitionRepository) Create(ctx context.Context, transition *domain.OrderStateTransition) error {
	query := `
		INSERT INTO order_state_transitions (
			order_id, from_status, to_status, triggered_by, admin_id, reason, metadata, ip_address
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at
	`

	return r.db.QueryRowContext(
		ctx, query,
		transition.OrderID, transition.FromStatus, transition.ToStatus,
		transition.TriggeredBy, transition.AdminID, transition.Reason,
		transition.Metadata, transition.IPAddress,
	).Scan(&transition.ID, &transition.CreatedAt)
}

func (r *OrderStateTransitionRepository) GetByOrderID(ctx context.Context, orderID int64) ([]*domain.OrderStateTransition, error) {
	var transitions []*domain.OrderStateTransition
	query := `SELECT * FROM order_state_transitions WHERE order_id = $1 ORDER BY created_at ASC`
	err := r.db.SelectContext(ctx, &transitions, query, orderID)
	return transitions, err
}