package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/merraki/merraki-backend/internal/domain"
	"github.com/merraki/merraki-backend/internal/pkg/logger"
	"github.com/merraki/merraki-backend/internal/repository"
	"github.com/merraki/merraki-backend/internal/service"
	"go.uber.org/zap"
)

// ============================================================================
// JOB PROCESSOR - Main background worker
// ============================================================================

type JobProcessor struct {
	jobRepo           repository.BackgroundJobRepository
	orderRepo         repository.OrderRepository
	orderItemRepo     repository.OrderItemRepository
	paymentRepo       repository.PaymentRepository
	webhookRepo       repository.PaymentWebhookRepository
	downloadTokenRepo repository.DownloadTokenRepository
	idempotencyRepo   repository.IdempotencyKeyRepository

	emailService     *service.EmailService
	downloadTokenSvc *service.DownloadTokenService
	paymentService   *service.PaymentService
	pdfService       *service.PDFService
	storageService   *service.StorageService

	workerID       string
	maxConcurrency int
	pollInterval   time.Duration
	shutdownChan   chan struct{}
}

func NewJobProcessor(
	jobRepo repository.BackgroundJobRepository,
	orderRepo repository.OrderRepository,
	orderItemRepo repository.OrderItemRepository,
	paymentRepo repository.PaymentRepository,
	webhookRepo repository.PaymentWebhookRepository,
	downloadTokenRepo repository.DownloadTokenRepository,
	idempotencyRepo repository.IdempotencyKeyRepository,
	emailService *service.EmailService,
	downloadTokenSvc *service.DownloadTokenService,
	paymentService *service.PaymentService,
	pdfService *service.PDFService,
	storageService *service.StorageService,
	workerID string,
) *JobProcessor {
	return &JobProcessor{
		jobRepo:           jobRepo,
		orderRepo:         orderRepo,
		orderItemRepo:     orderItemRepo,
		paymentRepo:       paymentRepo,
		webhookRepo:       webhookRepo,
		downloadTokenRepo: downloadTokenRepo,
		idempotencyRepo:   idempotencyRepo,
		emailService:      emailService,
		downloadTokenSvc:  downloadTokenSvc,
		paymentService:    paymentService,
		pdfService:        pdfService,
		storageService:    storageService,
		workerID:          workerID,
		maxConcurrency:    5,
		pollInterval:      5 * time.Second,
		shutdownChan:      make(chan struct{}),
	}
}

// ============================================================================
// START WORKER
// ============================================================================

func (w *JobProcessor) Start(ctx context.Context) error {
	logger.Info("Starting job processor",
		zap.String("worker_id", w.workerID),
		zap.Int("max_concurrency", w.maxConcurrency),
	)

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	semaphore := make(chan struct{}, w.maxConcurrency)

	for {
		select {
		case <-ctx.Done():
			logger.Info("Job processor shutting down", zap.String("worker_id", w.workerID))
			return nil

		case <-w.shutdownChan:
			logger.Info("Job processor stopped", zap.String("worker_id", w.workerID))
			return nil

		case <-ticker.C:
			// Acquire semaphore slot
			select {
			case semaphore <- struct{}{}:
				go func() {
					defer func() { <-semaphore }()
					w.processNextJob(ctx)
				}()
			default:
				// All workers busy, skip this tick
			}
		}
	}
}

func (w *JobProcessor) Stop() {
	close(w.shutdownChan)
}

// ============================================================================
// PROCESS NEXT JOB
// ============================================================================

func (w *JobProcessor) processNextJob(ctx context.Context) {
	// Try to acquire a job with lock
	jobs, err := w.jobRepo.GetPending(ctx, 1)
	if err != nil {
		logger.Error("Failed to fetch pending jobs", zap.Error(err))
		return
	}

	if len(jobs) == 0 {
		return // No jobs available
	}

	job := jobs[0]

	// Try to acquire lock
	locked, err := w.jobRepo.AcquireLock(ctx, job.ID, w.workerID, int(5*time.Minute.Seconds()))
	if err != nil || !locked {
		return // Job already locked by another worker
	}

	logger.Info("Processing job",
		zap.Int64("job_id", job.ID),
		zap.String("job_type", job.JobType),
		zap.String("worker_id", w.workerID),
	)

	// Update status to running
	job.Status = domain.JobStatusRunning
	_ = w.jobRepo.UpdateStatus(ctx, job.ID, domain.JobStatusRunning)

	// Process the job
	err = w.executeJob(ctx, job)
	if err != nil {
		logger.Error("Job execution failed",
			zap.Int64("job_id", job.ID),
			zap.String("job_type", job.JobType),
			zap.Error(err),
		)

		// Check if we should retry
		if job.RetryCount < job.MaxRetries {
			_ = w.jobRepo.IncrementRetryCount(ctx, job.ID)
			logger.Info("Job scheduled for retry",
				zap.Int64("job_id", job.ID),
				zap.Int("retry_count", job.RetryCount+1),
			)
		} else {
			// Max retries reached, mark as failed
			_ = w.jobRepo.MarkAsFailed(ctx, job.ID, err.Error())
			logger.Error("Job failed after max retries",
				zap.Int64("job_id", job.ID),
				zap.String("job_type", job.JobType),
			)
		}
	} else {
		// Success
		_ = w.jobRepo.MarkAsCompleted(ctx, job.ID)
		logger.Info("Job completed successfully",
			zap.Int64("job_id", job.ID),
			zap.String("job_type", job.JobType),
		)
	}
}

// ============================================================================
// EXECUTE JOB - Route to handler
// ============================================================================

func (w *JobProcessor) executeJob(ctx context.Context, job *domain.BackgroundJob) error {
	switch job.JobType {
	case "send_order_received_email":
		return w.handleSendOrderReceivedEmail(ctx, job)
	case "send_order_confirmation_email":
		return w.handleSendOrderConfirmationEmail(ctx, job)

	case "send_order_approval_email":
		return w.handleSendOrderApprovalEmail(ctx, job)

	case "send_order_rejection_email":
		return w.handleSendOrderRejectionEmail(ctx, job)

	case "send_admin_review_notification":
		return w.handleSendAdminReviewNotification(ctx, job)

	case "generate_download_tokens":
		return w.handleGenerateDownloadTokens(ctx, job)

	case "process_webhook":
		return w.handleProcessWebhook(ctx, job)

	case "process_refund":
		return w.handleProcessRefund(ctx, job)

	case "cleanup_expired_tokens":
		return w.handleCleanupExpiredTokens(ctx, job)

	case "cleanup_idempotency_keys":
		return w.handleCleanupIdempotencyKeys(ctx, job)

	default:
		return fmt.Errorf("unknown job type: %s", job.JobType)
	}
}

// ============================================================================
// JOB HANDLERS - Email Jobs
// ============================================================================

func (w *JobProcessor) handleSendOrderConfirmationEmail(ctx context.Context, job *domain.BackgroundJob) error {
	orderID, err := w.getInt64FromPayload(job.Payload, "order_id")
	if err != nil {
		return err
	}

	// Get order
	order, err := w.orderRepo.FindByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}

	// Get order items
	items, err := w.orderItemRepo.GetByOrderID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to get order items: %w", err)
	}

	// Send email
	return w.emailService.SendOrderConfirmation(ctx, order, items)
}

func (w *JobProcessor) handleSendOrderApprovalEmail(ctx context.Context, job *domain.BackgroundJob) error {
	orderID, err := w.getInt64FromPayload(job.Payload, "order_id")
	if err != nil {
		return err
	}

	// Get order
	order, err := w.orderRepo.FindByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}

	// Get download tokens
	tokens, err := w.downloadTokenRepo.GetByOrderID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to get download tokens: %w", err)
	}

	// Send email
	return w.emailService.SendOrderApproval(ctx, order, tokens)
}

func (w *JobProcessor) handleSendOrderRejectionEmail(ctx context.Context, job *domain.BackgroundJob) error {
	orderID, err := w.getInt64FromPayload(job.Payload, "order_id")
	if err != nil {
		return err
	}

	reason, _ := w.getStringFromPayload(job.Payload, "reason")

	// Get order
	order, err := w.orderRepo.FindByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}

	// Send email
	return w.emailService.SendOrderRejection(ctx, order, reason)
}

func (w *JobProcessor) handleSendAdminReviewNotification(ctx context.Context, job *domain.BackgroundJob) error {
	orderID, err := w.getInt64FromPayload(job.Payload, "order_id")
	if err != nil {
		return err
	}

	// Get order
	order, err := w.orderRepo.FindByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}

	// Send email to admin
	return w.emailService.SendAdminOrderNotification(ctx, order)
}

// ============================================================================
// JOB HANDLERS - Download Tokens
// ============================================================================

func (w *JobProcessor) handleGenerateDownloadTokens(ctx context.Context, job *domain.BackgroundJob) error {
	orderID, err := w.getInt64FromPayload(job.Payload, "order_id")
	if err != nil {
		return err
	}

	// Generate tokens
	if err := w.downloadTokenSvc.GenerateTokensForOrder(ctx, orderID); err != nil {
		return fmt.Errorf("failed to generate download tokens: %w", err)
	}

	logger.Info("Download tokens generated",
		zap.Int64("order_id", orderID),
	)

	return nil
}

// ============================================================================
// JOB HANDLERS - Payment Webhooks
// ============================================================================

func (w *JobProcessor) handleProcessWebhook(ctx context.Context, job *domain.BackgroundJob) error {
	webhookID, err := w.getInt64FromPayload(job.Payload, "webhook_id")
	if err != nil {
		return err
	}

	// Get webhook
	webhook, err := w.webhookRepo.FindByID(ctx, webhookID)
	if err != nil {
		return fmt.Errorf("failed to get webhook: %w", err)
	}

	// Parse webhook payload
	webhookData := webhook.Payload

	// Extract event type
	eventType := webhook.EventType

	// Handle different webhook events
	switch eventType {
	case "payment.captured":
		return w.handlePaymentCapturedWebhook(ctx, webhook, webhookData)

	case "payment.failed":
		return w.handlePaymentFailedWebhook(ctx, webhook, webhookData)

	case "refund.processed":
		return w.handleRefundProcessedWebhook(ctx, webhook, webhookData)

	default:
		logger.Warn("Unknown webhook event type",
			zap.String("event_type", eventType),
		)
		return nil // Not an error, just unknown event
	}
}

func (w *JobProcessor) handleSendOrderReceivedEmail(ctx context.Context, job *domain.BackgroundJob) error {
	orderID, err := w.getInt64FromPayload(job.Payload, "order_id")
	if err != nil {
		return err
	}

	// Get order
	order, err := w.orderRepo.FindByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}

	// Get order items
	items, err := w.orderItemRepo.GetByOrderID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to get order items: %w", err)
	}

	// Generate receipt PDF
	var pdfBytes []byte
	if w.pdfService != nil {
		pdfBytes, err = w.pdfService.GenerateOrderInvoice(ctx, order, items)
		if err != nil {
			// Log but don't fail — send email without PDF
			logger.Warn("Failed to generate receipt PDF, sending email without attachment",
				zap.Int64("order_id", orderID),
				zap.Error(err),
			)
			pdfBytes = nil
		}
	}

	// Send email to customer with order details + PDF receipt
	return w.emailService.SendOrderReceived(ctx, order, items, pdfBytes)
}

func (w *JobProcessor) handlePaymentCapturedWebhook(ctx context.Context, webhook *domain.PaymentWebhook, data map[string]interface{}) error {
	// Extract payment ID
	paymentID := webhook.GatewayPaymentID
	if paymentID == nil {
		return fmt.Errorf("webhook missing payment ID")
	}

	// Update payment status
	payment, err := w.paymentRepo.FindByGatewayPaymentID(ctx, *paymentID)
	if err != nil {
		return fmt.Errorf("failed to find payment: %w", err)
	}

	payment.Status = domain.PaymentStatusCaptured
	now := time.Now()
	payment.CapturedAt = &now

	if err := w.paymentRepo.Update(ctx, payment); err != nil {
		return fmt.Errorf("failed to update payment: %w", err)
	}

	logger.Info("Payment webhook processed",
		zap.String("payment_id", *paymentID),
		zap.String("event", "captured"),
	)

	return nil
}

func (w *JobProcessor) handlePaymentFailedWebhook(ctx context.Context, webhook *domain.PaymentWebhook, data map[string]interface{}) error {
	paymentID := webhook.GatewayPaymentID
	if paymentID == nil {
		return fmt.Errorf("webhook missing payment ID")
	}

	payment, err := w.paymentRepo.FindByGatewayPaymentID(ctx, *paymentID)
	if err != nil {
		return fmt.Errorf("failed to find payment: %w", err)
	}

	payment.Status = domain.PaymentStatusFailed

	if err := w.paymentRepo.Update(ctx, payment); err != nil {
		return fmt.Errorf("failed to update payment: %w", err)
	}

	logger.Info("Payment webhook processed",
		zap.String("payment_id", *paymentID),
		zap.String("event", "failed"),
	)

	return nil
}

func (w *JobProcessor) handleRefundProcessedWebhook(ctx context.Context, webhook *domain.PaymentWebhook, data map[string]interface{}) error {
	// Handle refund webhook
	logger.Info("Refund webhook processed")
	return nil
}

// ============================================================================
// JOB HANDLERS - Refunds
// ============================================================================

func (w *JobProcessor) handleProcessRefund(ctx context.Context, job *domain.BackgroundJob) error {
	orderID, err := w.getInt64FromPayload(job.Payload, "order_id")
	if err != nil {
		return err
	}

	reason, _ := w.getStringFromPayload(job.Payload, "reason")

	// Get order
	order, err := w.orderRepo.FindByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}

	// Get payment
	if order.GatewayPaymentID == nil {
		return fmt.Errorf("order has no payment ID")
	}

	payment, err := w.paymentRepo.FindByGatewayPaymentID(ctx, *order.GatewayPaymentID)
	if err != nil {
		return fmt.Errorf("failed to get payment: %w", err)
	}

	// Process refund via Razorpay
	refund, err := w.paymentService.CreateRefund(ctx, &service.CreateRefundRequest{
		PaymentID: *payment.GatewayPaymentID,
		Notes: map[string]string{
			"order_id": order.OrderNumber,
			"reason":   reason,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create refund: %w", err)
	}

	// Update payment status
	payment.Status = domain.PaymentStatusRefunded
	now := time.Now()
	payment.RefundedAt = &now

	if err := w.paymentRepo.Update(ctx, payment); err != nil {
		return fmt.Errorf("failed to update payment: %w", err)
	}

	// Update order status
	order.Status = domain.OrderStatusRefunded
	order.RefundedAt = &now

	if err := w.orderRepo.Update(ctx, order); err != nil {
		return fmt.Errorf("failed to update order: %w", err)
	}

	logger.Info("Refund processed",
		zap.String("order_number", order.OrderNumber),
		zap.String("refund_id", refund.ID),
	)

	return nil
}

// ============================================================================
// JOB HANDLERS - Cleanup Jobs
// ============================================================================

func (w *JobProcessor) handleCleanupExpiredTokens(ctx context.Context, job *domain.BackgroundJob) error {
	count, err := w.downloadTokenRepo.CleanupExpired(ctx)
	if err != nil {
		return fmt.Errorf("failed to cleanup expired tokens: %w", err)
	}

	logger.Info("Expired tokens cleaned up",
		zap.Int64("count", count),
	)

	return nil
}

func (w *JobProcessor) handleCleanupIdempotencyKeys(ctx context.Context, job *domain.BackgroundJob) error {
	count, err := w.idempotencyRepo.CleanupExpired(ctx)
	if err != nil {
		return fmt.Errorf("failed to cleanup idempotency keys: %w", err)
	}

	logger.Info("Expired idempotency keys cleaned up",
		zap.Int64("count", count),
	)

	return nil
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func (w *JobProcessor) getInt64FromPayload(payload domain.JSONMap, key string) (int64, error) {
	val, ok := payload[key]
	if !ok {
		return 0, fmt.Errorf("missing key in payload: %s", key)
	}

	// Handle both float64 (from JSON) and int64
	switch v := val.(type) {
	case float64:
		return int64(v), nil
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	default:
		return 0, fmt.Errorf("invalid type for key %s: %T", key, val)
	}
}

func (w *JobProcessor) getStringFromPayload(payload domain.JSONMap, key string) (string, error) {
	val, ok := payload[key]
	if !ok {
		return "", fmt.Errorf("missing key in payload: %s", key)
	}

	str, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("invalid type for key %s: expected string, got %T", key, val)
	}

	return str, nil
}
