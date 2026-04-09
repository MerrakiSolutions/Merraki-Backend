package repository

import (
	"context"

	"github.com/merraki/merraki-backend/internal/domain"
)

// ============================================================================
// REPOSITORY INTERFACES - Dependency Inversion
// ============================================================================

type CategoryRepository interface {
	Create(ctx context.Context, category *domain.Category) error
	FindByID(ctx context.Context, id int64) (*domain.Category, error)
	FindBySlug(ctx context.Context, slug string) (*domain.Category, error)
	GetAll(ctx context.Context, activeOnly bool) ([]*domain.Category, error)
	Update(ctx context.Context, category *domain.Category) error
	Delete(ctx context.Context, id int64) error
}

type TemplateRepository interface {
	// Core CRUD
	Create(ctx context.Context, template *domain.Template) error
	FindByID(ctx context.Context, id int64) (*domain.Template, error)
	FindBySlug(ctx context.Context, slug string) (*domain.Template, error)
	GetAll(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*domain.Template, int, error)
	GetAllWithRelations(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*domain.TemplateWithRelations, int, error)
	Update(ctx context.Context, template *domain.Template) error
	Delete(ctx context.Context, id int64) error

	// Stock management
	DecrementStock(ctx context.Context, id int64, quantity int) error
	IncrementDownloads(ctx context.Context, id int64) error
	IncrementViews(ctx context.Context, id int64) error

	// Images
	CreateImage(ctx context.Context, image *domain.TemplateImage) error
	GetImages(ctx context.Context, templateID int64) ([]*domain.TemplateImage, error)
	DeleteImage(ctx context.Context, id int64) error

	// Features
	CreateFeature(ctx context.Context, feature *domain.TemplateFeature) error
	GetFeatures(ctx context.Context, templateID int64) ([]*domain.TemplateFeature, error)
	DeleteFeature(ctx context.Context, id int64) error

	// Tags
	AddTag(ctx context.Context, templateID int64, tag string) error
	GetTags(ctx context.Context, templateID int64) ([]string, error)
	RemoveTags(ctx context.Context, templateID int64) error

	// Versions
	CreateVersion(ctx context.Context, version *domain.TemplateVersion) error
	GetVersions(ctx context.Context, templateID int64) ([]*domain.TemplateVersion, error)
	GetCurrentVersion(ctx context.Context, templateID int64) (*domain.TemplateVersion, error)
	SetCurrentVersion(ctx context.Context, templateID int64, versionID int64) error
}

type OrderRepository interface {
	// Core CRUD
	Create(ctx context.Context, order *domain.Order) error
	FindByID(ctx context.Context, id int64) (*domain.Order, error)
	FindByOrderNumber(ctx context.Context, orderNumber string) (*domain.Order, error)
	FindByGatewayOrderID(ctx context.Context, gatewayOrderID string) (*domain.Order, error)
	FindByEmail(ctx context.Context, email string, limit, offset int) ([]*domain.Order, int, error)
	GetAll(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*domain.Order, int, error)
	Update(ctx context.Context, order *domain.Order) error

	// With items
	GetWithItems(ctx context.Context, id int64) (*domain.OrderWithItems, error)

	// State management
	UpdateStatus(ctx context.Context, id int64, newStatus domain.OrderStatus, adminID *int64) error

	// Admin actions
	Approve(ctx context.Context, id int64, adminID int64, notes *string) error
	Reject(ctx context.Context, id int64, adminID int64, reason string) error

	// Analytics
	GetRevenueByDateRange(ctx context.Context, startDate, endDate string) (float64, error)
	GetOrderCountByStatus(ctx context.Context) (map[domain.OrderStatus]int, error)

	// MarkasPaid
	MarkAsPaid(ctx context.Context, id int64, adminID int64, gatewayOrderID string) error

	// Delete
	Delete(ctx context.Context, id int64, adminID int64) error
}

type OrderItemRepository interface {
	Create(ctx context.Context, item *domain.OrderItem) error
	CreateBatch(ctx context.Context, items []*domain.OrderItem) error
	GetByOrderID(ctx context.Context, orderID int64) ([]*domain.OrderItem, error)
	GetByID(ctx context.Context, id int64) (*domain.OrderItem, error)
	IncrementDownloadCount(ctx context.Context, id int64) error
}

type PaymentRepository interface {
	Create(ctx context.Context, payment *domain.Payment) error
	FindByID(ctx context.Context, id int64) (*domain.Payment, error)
	FindByGatewayOrderID(ctx context.Context, gatewayOrderID string) (*domain.Payment, error)
	FindByGatewayPaymentID(ctx context.Context, gatewayPaymentID string) (*domain.Payment, error)
	GetByOrderID(ctx context.Context, orderID int64) ([]*domain.Payment, error)
	Update(ctx context.Context, payment *domain.Payment) error
	UpdateStatus(ctx context.Context, id int64, status domain.PaymentStatus) error
	MarkAsVerified(ctx context.Context, id int64) error
}

type PaymentWebhookRepository interface {
	Create(ctx context.Context, webhook *domain.PaymentWebhook) error
	FindByID(ctx context.Context, id int64) (*domain.PaymentWebhook, error)
	GetUnprocessed(ctx context.Context, limit int) ([]*domain.PaymentWebhook, error)
	MarkAsProcessed(ctx context.Context, id int64) error
	IncrementRetryCount(ctx context.Context, id int64, errorMsg string) error
}

type DownloadTokenRepository interface {
	Create(ctx context.Context, token *domain.DownloadToken) error
	FindByToken(ctx context.Context, token string) (*domain.DownloadToken, error)
	GetByOrderID(ctx context.Context, orderID int64) ([]*domain.DownloadToken, error)
	GetByEmail(ctx context.Context, email string) ([]*domain.DownloadToken, error)
	IncrementDownloadCount(ctx context.Context, id int64) error
	Revoke(ctx context.Context, id int64, adminID int64, reason string) error
	CleanupExpired(ctx context.Context) (int64, error)
}

type DownloadRepository interface {
	Create(ctx context.Context, download *domain.Download) error
	GetByTokenID(ctx context.Context, tokenID int64) ([]*domain.Download, error)
	GetByOrderID(ctx context.Context, orderID int64) ([]*domain.Download, error)
	MarkAsCompleted(ctx context.Context, id int64, durationMS int) error
	MarkAsFailed(ctx context.Context, id int64, errorMsg string) error
}

type OrderStateTransitionRepository interface {
	Create(ctx context.Context, transition *domain.OrderStateTransition) error
	GetByOrderID(ctx context.Context, orderID int64) ([]*domain.OrderStateTransition, error)
}

type IdempotencyKeyRepository interface {
	Create(ctx context.Context, key *domain.IdempotencyKey) error
	FindByKey(ctx context.Context, key string) (*domain.IdempotencyKey, error)
	CleanupExpired(ctx context.Context) (int64, error)
}

type BackgroundJobRepository interface {
	Create(ctx context.Context, job *domain.BackgroundJob) error
	FindByID(ctx context.Context, id int64) (*domain.BackgroundJob, error)
	GetPending(ctx context.Context, limit int) ([]*domain.BackgroundJob, error)
	AcquireLock(ctx context.Context, id int64, workerID string, lockDuration int) (bool, error)
	UpdateStatus(ctx context.Context, id int64, status domain.JobStatus) error
	MarkAsCompleted(ctx context.Context, id int64) error
	MarkAsFailed(ctx context.Context, id int64, errorMsg string) error
	IncrementRetryCount(ctx context.Context, id int64) error
	GetJobsByType(ctx context.Context, jobType string, limit int) ([]*domain.BackgroundJob, error)
}

type CircuitBreakerRepository interface {
	GetByServiceName(ctx context.Context, serviceName string) (*domain.CircuitBreakerState, error)
	UpdateState(ctx context.Context, state *domain.CircuitBreakerState) error
	IncrementFailure(ctx context.Context, serviceName string) error
	IncrementSuccess(ctx context.Context, serviceName string) error
	ResetCounts(ctx context.Context, serviceName string) error
}
