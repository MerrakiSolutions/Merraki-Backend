// internal/repository/activity_log.go
package repository	

import (
	"context"
	"github.com/merraki/merraki-backend/internal/domain"
)

type ActivityLogRepository interface {
	Create(ctx context.Context, log *domain.ActivityLog) error
	GetAll(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*domain.ActivityLog, int, error)
}