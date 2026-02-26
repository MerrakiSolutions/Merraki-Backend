package postgres

import (
	"context"
	"fmt"

	"github.com/merraki/merraki-backend/internal/domain"
)

type ActivityLogRepository struct {
	db *Database
}

func NewActivityLogRepository(db *Database) *ActivityLogRepository {
	return &ActivityLogRepository{db: db}
}

func (r *ActivityLogRepository) Create(ctx context.Context, log *domain.ActivityLog) error {
	query := `
		INSERT INTO activity_logs (admin_id, action, entity_type, entity_id, details, ip_address)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`

	return r.db.DB.QueryRowContext(
		ctx, query,
		log.AdminID, log.Action, log.EntityType, log.EntityID, log.Details, log.IPAddress,
	).Scan(&log.ID, &log.CreatedAt)
}

func (r *ActivityLogRepository) GetAll(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*domain.ActivityLog, int, error) {
	var logs []*domain.ActivityLog
	
	query := `SELECT * FROM activity_logs WHERE 1=1`
	countQuery := `SELECT COUNT(*) FROM activity_logs WHERE 1=1`
	args := []interface{}{}
	argCount := 1

	if adminID, ok := filters["admin_id"].(int64); ok && adminID > 0 {
		query += fmt.Sprintf(" AND admin_id = $%d", argCount)
		countQuery += fmt.Sprintf(" AND admin_id = $%d", argCount)
		args = append(args, adminID)
		argCount++
	}

	if action, ok := filters["action"].(string); ok && action != "" {
		query += fmt.Sprintf(" AND action = $%d", argCount)
		countQuery += fmt.Sprintf(" AND action = $%d", argCount)
		args = append(args, action)
		argCount++
	}

	query += " ORDER BY created_at DESC"
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argCount, argCount+1)

	var total int
	if err := r.db.DB.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	if err := r.db.DB.SelectContext(ctx, &logs, query, args...); err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}