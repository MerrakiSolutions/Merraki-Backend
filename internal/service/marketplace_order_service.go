package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/merraki/merraki-backend/internal/domain"
	"github.com/merraki/merraki-backend/internal/pkg/logger"
	"github.com/merraki/merraki-backend/internal/repository"
	"go.uber.org/zap"
)

// ============================================================================
// ORDER SERVICE
// ============================================================================

type OrderService struct {
	orderRepo       repository.OrderRepository
	orderItemRepo   repository.OrderItemRepository
	templateRepo    repository.TemplateRepository
	paymentRepo     repository.PaymentRepository
	idempotencyRepo repository.IdempotencyKeyRepository
	transitionRepo  repository.OrderStateTransitionRepository
	activityLogRepo repository.ActivityLogRepository
	paymentService  *PaymentService
	emailService    *EmailService
	jobRepo         repository.BackgroundJobRepository
}

func NewOrderService(
	orderRepo repository.OrderRepository,
	orderItemRepo repository.OrderItemRepository,
	templateRepo repository.TemplateRepository,
	paymentRepo repository.PaymentRepository,
	idempotencyRepo repository.IdempotencyKeyRepository,
	transitionRepo repository.OrderStateTransitionRepository,
	activityLogRepo repository.ActivityLogRepository,
	paymentService *PaymentService,
	emailService *EmailService,
	jobRepo repository.BackgroundJobRepository,
) *OrderService {
	return &OrderService{
		orderRepo:       orderRepo,
		orderItemRepo:   orderItemRepo,
		templateRepo:    templateRepo,
		paymentRepo:     paymentRepo,
		idempotencyRepo: idempotencyRepo,
		transitionRepo:  transitionRepo,
		activityLogRepo: activityLogRepo,
		paymentService:  paymentService,
		emailService:    emailService,
		jobRepo:         jobRepo,
	}
}

// ============================================================================
// CREATE ORDER - Guest Checkout (server-side price authority)
// ============================================================================

type CreateOrderRequest struct {
	CustomerEmail     string                 `json:"customer_email" validate:"required,email"`
	CustomerName      string                 `json:"customer_name" validate:"required"`
	CustomerPhone     string                 `json:"customer_phone"`
	BillingAddress    *domain.BillingAddress `json:"billing_address"`
	Items             []CreateOrderItem      `json:"items" validate:"required,min=1,dive"`
	IdempotencyKey    string                 `json:"idempotency_key"`
	CustomerIP        string                 `json:"-"`
	CustomerUserAgent string                 `json:"-"`
}

type CreateOrderItem struct {
	TemplateID int64 `json:"template_id" validate:"required"`
}

func (s *OrderService) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*domain.Order, error) {
	// 1. Check idempotency
	if req.IdempotencyKey != "" {
		existingKey, err := s.idempotencyRepo.FindByKey(ctx, req.IdempotencyKey)
		if err == nil && existingKey != nil {
			logger.Info("Idempotent create_order request detected", zap.String("key", req.IdempotencyKey))
			if existingKey.EntityID != nil {
				order, err := s.orderRepo.FindByID(ctx, *existingKey.EntityID)
				if err == nil && order != nil {
					return order, nil
				}
			}
		}
	}

	// 2. Validate templates and build order items (server-side pricing authority)
	var orderItems []*domain.OrderItem
	var subtotalCents int64

	for _, item := range req.Items {
		template, err := s.templateRepo.FindByID(ctx, item.TemplateID)
		if err != nil || template == nil {
			return nil, fmt.Errorf("template %d not found", item.TemplateID)
		}

		if !template.IsAvailable() {
			return nil, fmt.Errorf("template %d is not available for purchase", item.TemplateID)
		}

		priceCents := template.GetCurrentPriceCents()

		orderItems = append(orderItems, &domain.OrderItem{
			TemplateID:      template.ID,
			TemplateName:    template.Name,
			TemplateSlug:    template.Slug,
			TemplateVersion: template.CurrentVersion,
			PriceUSDCents:   priceCents,
			FileURL:         template.FileURL,
			FileFormat:      template.FileFormat,
			FileSizeMB:      template.FileSizeMB,
		})

		subtotalCents += priceCents
	}

	// 3. Calculate totals (all in cents)
	taxCents := int64(0)
	discountCents := int64(0)
	totalCents := subtotalCents + taxCents - discountCents

	// 4. Build order
	order := &domain.Order{
		OrderNumber:            s.generateOrderNumber(),
		CustomerEmail:          req.CustomerEmail,
		CustomerName:           req.CustomerName,
		CustomerPhone:          &req.CustomerPhone,
		CustomerIP:             &req.CustomerIP,
		CustomerUserAgent:      &req.CustomerUserAgent,
		SubtotalUSDCents:       subtotalCents,
		TaxAmountUSDCents:      taxCents,
		DiscountAmountUSDCents: discountCents,
		TotalAmountUSDCents:    totalCents,
		PaymentGateway:         "razorpay",
		Status:                 domain.OrderStatusPending,
		Metadata:               make(domain.JSONMap),
	}

	if req.IdempotencyKey != "" {
		order.IdempotencyKey = &req.IdempotencyKey
	}

	if req.BillingAddress != nil {
		order.BillingName = &req.BillingAddress.Name
		order.BillingEmail = &req.BillingAddress.Email
		order.BillingPhone = &req.BillingAddress.Phone
		order.BillingAddressLine1 = &req.BillingAddress.AddressLine1
		order.BillingAddressLine2 = &req.BillingAddress.AddressLine2
		order.BillingCity = &req.BillingAddress.City
		order.BillingState = &req.BillingAddress.State
		order.BillingCountry = req.BillingAddress.Country
		order.BillingPostalCode = &req.BillingAddress.PostalCode
	}

	// 5. Save order
	if err := s.orderRepo.Create(ctx, order); err != nil {
		return nil, err
	}

	// 6. Save order items
	for _, item := range orderItems {
		item.OrderID = order.ID
	}
	if err := s.orderItemRepo.CreateBatch(ctx, orderItems); err != nil {
		return nil, err
	}

	// 7. Save idempotency key
	if req.IdempotencyKey != "" {
		_ = s.idempotencyRepo.Create(ctx, &domain.IdempotencyKey{
			Key:           req.IdempotencyKey,
			OperationType: "create_order",
			EntityType:    strPtr("order"),
			EntityID:      &order.ID,
			ExpiresAt:     time.Now().Add(24 * time.Hour),
		})
	}

	s.logActivity(ctx, "order_created", order.ID, 0, map[string]interface{}{
		"order_number":    order.OrderNumber,
		"total_usd_cents": order.TotalAmountUSDCents,
	})

	logger.Info("Order created",
		zap.String("order_number", order.OrderNumber),
		zap.Int64("total_cents", order.TotalAmountUSDCents),
	)

	return order, nil
}

// ============================================================================
// INITIATE PAYMENT - Create Razorpay order
// ============================================================================

func (s *OrderService) InitiatePayment(ctx context.Context, orderID int64) (*domain.Payment, error) {
	order, err := s.orderRepo.FindByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, domain.ErrNotFound
	}

	// Allow re-initiation if already payment_initiated (handles page refresh)
	if order.Status != domain.OrderStatusPending &&
		order.Status != domain.OrderStatusPaymentInitiated {
		return nil, fmt.Errorf("order cannot initiate payment in state: %s", order.Status)
	}

	// Re-use existing gateway order if already initiated
	if order.Status == domain.OrderStatusPaymentInitiated && order.GatewayOrderID != nil {
		payment, err := s.paymentRepo.FindByGatewayOrderID(ctx, *order.GatewayOrderID)
		if err == nil && payment != nil {
			logger.Info("Re-using existing Razorpay order",
				zap.String("order_number", order.OrderNumber),
				zap.String("gateway_order_id", *order.GatewayOrderID),
			)
			return payment, nil
		}
	}

	// Create Razorpay order — Razorpay expects amount in smallest currency unit (paise for INR)
	// Since we're USD-only, amount is in cents
	razorpayOrder, err := s.paymentService.CreateOrder(ctx, &CreateRazorpayOrderRequest{
		AmountUSDCents: order.TotalAmountUSDCents,
		Receipt:        order.OrderNumber,
		Notes: map[string]string{
			"order_id":       fmt.Sprintf("%d", order.ID),
			"order_number":   order.OrderNumber,
			"customer_email": order.CustomerEmail,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create razorpay order: %w", err)
	}

	// Update order
	order.GatewayOrderID = &razorpayOrder.ID
	order.Status = domain.OrderStatusPaymentInitiated
	if err := s.orderRepo.Update(ctx, order); err != nil {
		return nil, err
	}

	// Create payment record
	payment := &domain.Payment{
		OrderID:        order.ID,
		Gateway:        "razorpay",
		GatewayOrderID: razorpayOrder.ID,
		AmountUSDCents: order.TotalAmountUSDCents,
		Status:         domain.PaymentStatusCreated,
		GatewayResponse: domain.JSONMap{
			"order_id": razorpayOrder.ID,
			"amount":   razorpayOrder.Amount,
		},
	}

	if err := s.paymentRepo.Create(ctx, payment); err != nil {
		return nil, err
	}

	s.logActivity(ctx, "payment_initiated", order.ID, 0, map[string]interface{}{
		"gateway_order_id": razorpayOrder.ID,
		"amount_cents":     order.TotalAmountUSDCents,
	})

	logger.Info("Payment initiated",
		zap.String("order_number", order.OrderNumber),
		zap.String("gateway_order_id", razorpayOrder.ID),
	)

	return payment, nil
}

// ============================================================================
// VERIFY PAYMENT - Signature verification
// ============================================================================

type VerifyPaymentRequest struct {
	OrderID           int64  `json:"order_id" validate:"required"`
	RazorpayOrderID   string `json:"razorpay_order_id" validate:"required"`
	RazorpayPaymentID string `json:"razorpay_payment_id" validate:"required"`
	RazorpaySignature string `json:"razorpay_signature" validate:"required"`
	IdempotencyKey    string `json:"idempotency_key"`
}

func (s *OrderService) VerifyPayment(ctx context.Context, req *VerifyPaymentRequest) (*domain.Order, error) {
	// 1. Check idempotency
	if req.IdempotencyKey != "" {
		existingKey, err := s.idempotencyRepo.FindByKey(ctx, req.IdempotencyKey)
		if err == nil && existingKey != nil {
			logger.Info("Duplicate payment verification detected", zap.String("key", req.IdempotencyKey))
			if existingKey.EntityID != nil {
				order, err := s.orderRepo.FindByID(ctx, *existingKey.EntityID)
				if err == nil && order != nil {
					return order, nil
				}
			}
		}
	}

	// 2. Get order
	order, err := s.orderRepo.FindByID(ctx, req.OrderID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, domain.ErrNotFound
	}

	if order.Status != domain.OrderStatusPaymentInitiated &&
		order.Status != domain.OrderStatusPaymentProcessing {
		return nil, fmt.Errorf("invalid order state for payment verification: %s", order.Status)
	}

	// 3. Get payment record
	payment, err := s.paymentRepo.FindByGatewayOrderID(ctx, req.RazorpayOrderID)
	if err != nil || payment == nil {
		return nil, fmt.Errorf("payment record not found for gateway order: %s", req.RazorpayOrderID)
	}

	// 4. Verify Razorpay signature
	isValid := s.paymentService.VerifyPaymentSignature(
		req.RazorpayOrderID,
		req.RazorpayPaymentID,
		req.RazorpaySignature,
	)

	if !isValid {
		// Mark order and payment as failed
		order.Status = domain.OrderStatusFailed
		_ = s.orderRepo.Update(ctx, order)

		payment.Status = domain.PaymentStatusFailed
		payment.ErrorCode = strPtr("SIGNATURE_VERIFICATION_FAILED")
		payment.ErrorDescription = strPtr("Payment signature verification failed")
		payment.ErrorSource = strPtr("internal")
		_ = s.paymentRepo.Update(ctx, payment)

		s.logActivity(ctx, "payment_verification_failed", order.ID, 0, map[string]interface{}{
			"reason": "signature_mismatch",
		})

		return nil, fmt.Errorf("payment signature verification failed")
	}

	// 5. Update payment
	now := time.Now()
	payment.GatewayPaymentID = &req.RazorpayPaymentID
	payment.GatewaySignature = &req.RazorpaySignature
	payment.SignatureVerified = true
	payment.Status = domain.PaymentStatusCaptured
	payment.VerifiedAt = &now
	payment.CapturedAt = &now
	if err := s.paymentRepo.Update(ctx, payment); err != nil {
		return nil, err
	}

	// 6. Update order — always send to admin_review
	order.GatewayPaymentID = &req.RazorpayPaymentID
	order.Status = domain.OrderStatusAdminReview
	if err := s.orderRepo.Update(ctx, order); err != nil {
		return nil, err
	}

	// 7. Save idempotency key
	if req.IdempotencyKey != "" {
		_ = s.idempotencyRepo.Create(ctx, &domain.IdempotencyKey{
			Key:           req.IdempotencyKey,
			OperationType: "verify_payment",
			EntityType:    strPtr("order"),
			EntityID:      &order.ID,
			ExpiresAt:     time.Now().Add(24 * time.Hour),
		})
	}

	// 8. Enqueue background jobs
	_ = s.enqueueJob(ctx, "send_order_received_email", map[string]interface{}{
		"order_id": order.ID,
	})
	_ = s.enqueueJob(ctx, "send_admin_review_notification", map[string]interface{}{
		"order_id":     order.ID,
		"order_number": order.OrderNumber,
		"amount_cents": order.TotalAmountUSDCents,
		"customer":     order.CustomerEmail,
	})

	s.logActivity(ctx, "payment_verified", order.ID, 0, map[string]interface{}{
		"payment_id":   payment.ID,
		"amount_cents": payment.AmountUSDCents,
		"status":       order.Status,
	})

	logger.Info("Payment verified",
		zap.String("order_number", order.OrderNumber),
		zap.String("payment_id", req.RazorpayPaymentID),
	)

	return order, nil
}

// ============================================================================
// WEBHOOK HANDLER - Razorpay payment events
// ============================================================================

// MarkPaymentCaptured

func (s *OrderService) MarkPaymentCaptured(
	ctx context.Context,
	gatewayOrderID, gatewayPaymentID string,
) error {

	// 1. Find payment by gateway order
	payment, err := s.paymentRepo.FindByGatewayOrderID(ctx, gatewayOrderID)
	if err != nil || payment == nil {
		return fmt.Errorf("payment record not found for gateway order: %s", gatewayOrderID)
	}

	// 2. IDEMPOTENCY GUARD (CRITICAL)
	if payment.Status == domain.PaymentStatusCaptured {
		logger.Info("Duplicate webhook ignored",
			zap.String("gateway_order_id", gatewayOrderID),
			zap.String("gateway_payment_id", gatewayPaymentID),
		)
		return nil
	}

	// 3. Update payment
	payment.Status = domain.PaymentStatusCaptured
	now := time.Now()
	payment.CapturedAt = &now
	payment.GatewayPaymentID = &gatewayPaymentID

	if err := s.paymentRepo.Update(ctx, payment); err != nil {
		return err
	}

	// 4. Load order
	order, err := s.orderRepo.FindByID(ctx, payment.OrderID)
	if err != nil || order == nil {
		return fmt.Errorf("order not found for payment: %d", payment.OrderID)
	}

	// 5. Only update if not already finalized
	if order.Status == domain.OrderStatusPending ||
		order.Status == domain.OrderStatusPaymentInitiated {

		order.Status = domain.OrderStatusPaid // ✅ FIXED
		order.GatewayPaymentID = &gatewayPaymentID

		if err := s.orderRepo.Update(ctx, order); err != nil {
			return err
		}

		s.logActivity(ctx, "payment_captured", order.ID, 0, map[string]interface{}{
			"payment_id": gatewayPaymentID,
		})

		logger.Info("Payment captured via webhook",
			zap.String("order_number", order.OrderNumber),
			zap.String("payment_id", gatewayPaymentID),
		)
	}

	return nil
}

func (s *OrderService) MarkPaymentFailed(
	ctx context.Context,
	gatewayOrderID string,
) error {

	// 1. Find payment by gateway order ID
	payment, err := s.paymentRepo.FindByGatewayOrderID(ctx, gatewayOrderID)
	if err != nil || payment == nil {
		return fmt.Errorf("payment record not found for gateway order: %s", gatewayOrderID)
	}

	// 2. IDEMPOTENCY GUARD (avoid duplicate webhook processing)
	if payment.Status == domain.PaymentStatusFailed {
		logger.Info("Duplicate payment.failed webhook ignored",
			zap.String("gateway_order_id", gatewayOrderID),
		)
		return nil
	}

	// 3. Update payment status
	payment.Status = domain.PaymentStatusFailed
	now := time.Now()
	payment.FailedAt = &now // make sure this field exists in domain
	payment.UpdatedAt = now // if you track updates

	if err := s.paymentRepo.Update(ctx, payment); err != nil {
		return err
	}

	// 4. Load order
	order, err := s.orderRepo.FindByID(ctx, payment.OrderID)
	if err != nil || order == nil {
		return fmt.Errorf("order not found for payment: %d", payment.OrderID)
	}

	// 5. Update order ONLY if still in active payment state
	if order.Status == domain.OrderStatusPending ||
		order.Status == domain.OrderStatusPaymentInitiated {

		order.Status = domain.OrderStatusFailed
		order.UpdatedAt = now

		if err := s.orderRepo.Update(ctx, order); err != nil {
			return err
		}

		s.logActivity(ctx, "payment_failed", order.ID, 0, map[string]interface{}{
			"gateway_order_id": gatewayOrderID,
		})

		logger.Warn("Payment failed via webhook",
			zap.String("order_number", order.OrderNumber),
			zap.String("gateway_order_id", gatewayOrderID),
		)
	}

	return nil
}

// ============================================================================
// ADMIN ACTIONS
// ============================================================================

func (s *OrderService) ApproveOrder(ctx context.Context, orderID int64, adminID int64, notes *string) error {
	if err := s.orderRepo.Approve(ctx, orderID, adminID, notes); err != nil {
		return err
	}

	order, err := s.orderRepo.FindByID(ctx, orderID)
	if err != nil || order == nil {
		return err
	}

	_ = s.enqueueJob(ctx, "generate_download_tokens", map[string]interface{}{
		"order_id": order.ID,
	})
	_ = s.enqueueJob(ctx, "send_order_confirmation_email", map[string]interface{}{
		"order_id":     order.ID,
		"order_number": order.OrderNumber,
		"customer":     order.CustomerEmail,
	})

	s.logActivity(ctx, "order_approved", order.ID, adminID, map[string]interface{}{
		"notes": notes,
	})

	logger.Info("Order approved",
		zap.String("order_number", order.OrderNumber),
		zap.Int64("admin_id", adminID),
	)

	return nil
}

func (s *OrderService) RejectOrder(ctx context.Context, orderID int64, adminID int64, reason string) error {
	if err := s.orderRepo.Reject(ctx, orderID, adminID, reason); err != nil {
		return err
	}

	order, err := s.orderRepo.FindByID(ctx, orderID)
	if err != nil || order == nil {
		return err
	}

	_ = s.enqueueJob(ctx, "process_refund", map[string]interface{}{
		"order_id": order.ID,
		"reason":   reason,
	})
	_ = s.enqueueJob(ctx, "send_order_rejection_email", map[string]interface{}{
		"order_id":     order.ID,
		"order_number": order.OrderNumber,
		"customer":     order.CustomerEmail,
		"reason":       reason,
	})

	s.logActivity(ctx, "order_rejected", order.ID, adminID, map[string]interface{}{
		"reason": reason,
	})

	logger.Info("Order rejected",
		zap.String("order_number", order.OrderNumber),
		zap.Int64("admin_id", adminID),
		zap.String("reason", reason),
	)

	return nil
}

func (s *OrderService) MarkOrderAsPaid(ctx context.Context, orderID int64, adminID int64, gatewayOrderID string) error {
	if err := s.orderRepo.MarkAsPaid(ctx, orderID, adminID, gatewayOrderID); err != nil {
		return err
	}

	order, err := s.orderRepo.FindByID(ctx, orderID)
	if err != nil || order == nil {
		return err
	}

	_ = s.enqueueJob(ctx, "generate_download_tokens", map[string]interface{}{
		"order_id": order.ID,
	})
	_ = s.enqueueJob(ctx, "send_order_confirmation_email", map[string]interface{}{
		"order_id":     order.ID,
		"order_number": order.OrderNumber,
		"customer":     order.CustomerEmail,
	})

	s.logActivity(ctx, "order_marked_as_paid", order.ID, 0, nil)

	logger.Info("Order marked as paid",
		zap.String("order_number", order.OrderNumber),
	)

	return nil
}

func (s *OrderService) DeleteOrder(ctx context.Context, orderID int64, adminID int64) error {
	if err := s.orderRepo.Delete(ctx, orderID, adminID); err != nil {
		return err
	}

	s.logActivity(ctx, "order_deleted", orderID, adminID, nil)

	logger.Info("Order deleted",
		zap.Int64("order_id", orderID),
		zap.Int64("admin_id", adminID),
	)

	return nil
}

// ============================================================================
// QUERY METHODS
// ============================================================================

func (s *OrderService) GetOrderByID(ctx context.Context, id int64) (*domain.OrderWithItems, error) {
	return s.orderRepo.GetWithItems(ctx, id)
}

func (s *OrderService) GetOrderByNumber(ctx context.Context, orderNumber string) (*domain.OrderWithItems, error) {
	order, err := s.orderRepo.FindByOrderNumber(ctx, orderNumber)
	if err != nil {
		return nil, err
	}
	return s.orderRepo.GetWithItems(ctx, order.ID)
}

func (s *OrderService) GetOrdersByEmail(ctx context.Context, email string, page, limit int) ([]*domain.Order, int, error) {
	return s.orderRepo.FindByEmail(ctx, email, limit, (page-1)*limit)
}

func (s *OrderService) GetAllOrders(ctx context.Context, filters map[string]interface{}, page, limit int) ([]*domain.Order, int, error) {
	return s.orderRepo.GetAll(ctx, filters, limit, (page-1)*limit)
}

func (s *OrderService) GetOrderTransitions(ctx context.Context, orderID int64) ([]*domain.OrderStateTransition, error) {
	return s.transitionRepo.GetByOrderID(ctx, orderID)
}

// ============================================================================
// HELPERS
// ============================================================================

func (s *OrderService) generateOrderNumber() string {
	return fmt.Sprintf("ORD-%s-%s", time.Now().Format("20060102"), generateRandomString(6))
}

func (s *OrderService) enqueueJob(ctx context.Context, jobType string, payload map[string]interface{}) error {
	return s.jobRepo.Create(ctx, &domain.BackgroundJob{
		JobType:     jobType,
		Payload:     domain.JSONMap(payload),
		Status:      domain.JobStatusPending,
		MaxRetries:  3,
		ScheduledAt: time.Now(),
		Priority:    0,
	})
}

func (s *OrderService) logActivity(ctx context.Context, action string, entityID int64, adminID int64, metadata map[string]interface{}) {
	if s.activityLogRepo == nil {
		return
	}

	jsonMetadata := make(domain.JSONMap)
	for k, v := range metadata {
		jsonMetadata[k] = v
	}

	entityType := "order"
	var adminIDPtr *int64
	if adminID > 0 {
		adminIDPtr = &adminID
	}

	_ = s.activityLogRepo.Create(ctx, &domain.ActivityLog{
		Action:     action,
		EntityType: &entityType,
		EntityID:   &entityID,
		AdminID:    adminIDPtr,
		Details:    jsonMetadata,
	})
}

func generateRandomString(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			b[i] = charset[i%len(charset)]
			continue
		}
		b[i] = charset[n.Int64()]
	}
	return string(b)
}
