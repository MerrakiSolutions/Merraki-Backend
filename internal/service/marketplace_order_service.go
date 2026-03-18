package service

import (
	"context"
	"fmt"
	"time"

	"github.com/merraki/merraki-backend/internal/domain"
	"github.com/merraki/merraki-backend/internal/pkg/logger"
	"github.com/merraki/merraki-backend/internal/repository"
	"go.uber.org/zap"
)

// ============================================================================
// ORDER SERVICE - Core marketplace business logic
// ============================================================================

type OrderService struct {
	orderRepo           repository.OrderRepository
	orderItemRepo       repository.OrderItemRepository
	templateRepo        repository.TemplateRepository
	paymentRepo         repository.PaymentRepository
	idempotencyRepo     repository.IdempotencyKeyRepository
	transitionRepo      repository.OrderStateTransitionRepository
	activityLogRepo     repository.ActivityLogRepository
	paymentService      *PaymentService
	emailService        *EmailService
	jobRepo             repository.BackgroundJobRepository
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
// CREATE ORDER - Guest Checkout (Server-side price authority)
// ============================================================================

type CreateOrderRequest struct {
	CustomerEmail    string                  `json:"customer_email" validate:"required,email"`
	CustomerName     string                  `json:"customer_name" validate:"required"`
	CustomerPhone    string                  `json:"customer_phone"`
	BillingAddress   *domain.BillingAddress  `json:"billing_address"`
	Items            []CreateOrderItem       `json:"items" validate:"required,min=1,dive"`
	Currency         string                  `json:"currency" validate:"required,oneof=INR USD"`
	IdempotencyKey   string                  `json:"idempotency_key" validate:"required"`
	CustomerIP       string                  `json:"-"` // From request context
	CustomerUserAgent string                 `json:"-"` // From request context
}

type CreateOrderItem struct {
	TemplateID int64 `json:"template_id" validate:"required"`
	Quantity   int   `json:"quantity" validate:"required,min=1"`
}

func (s *OrderService) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*domain.Order, error) {
	// 1. Check idempotency
	existingKey, err := s.idempotencyRepo.FindByKey(ctx, req.IdempotencyKey)
	if err == nil {
		// Idempotency key exists - return cached response
		logger.Info("Idempotent request detected", zap.String("key", req.IdempotencyKey))
		
		if existingKey.EntityID != nil {
			order, err := s.orderRepo.FindByID(ctx, *existingKey.EntityID)
			if err == nil {
				return order, nil
			}
		}
	}

	// 2. Validate and fetch templates (SERVER-SIDE PRICING AUTHORITY)
	var orderItems []*domain.OrderItem
	var subtotal float64

	for _, item := range req.Items {
		template, err := s.templateRepo.FindByID(ctx, item.TemplateID)
		if err != nil {
			return nil, fmt.Errorf("template %d not found", item.TemplateID)
		}

		// Check availability
		if !template.IsInStock() {
			return nil, domain.ErrInsufficientStock
		}

		// Get current price (SERVER DECIDES PRICE - NEVER TRUST CLIENT)
		unitPrice := template.GetCurrentPrice(req.Currency)
		itemSubtotal := unitPrice * float64(item.Quantity)

		orderItem := &domain.OrderItem{
			TemplateID:      template.ID,
			TemplateName:    template.Name,
			TemplateSlug:    template.Slug,
			TemplateVersion: template.CurrentVersion,
			UnitPrice:       unitPrice,
			Quantity:        item.Quantity,
			Subtotal:        itemSubtotal,
			FileURL:         template.FileURL,
			FileFormat:      template.FileFormat,
			FileSizeMB:      template.FileSizeMB,
		}

		orderItems = append(orderItems, orderItem)
		subtotal += itemSubtotal
	}

	// 3. Calculate totals
	taxAmount := 0.0 // TODO: Implement tax calculation based on billing address
	discountAmount := 0.0
	totalAmount := subtotal + taxAmount - discountAmount

	// 4. Generate order number
	orderNumber := s.generateOrderNumber()

	// 5. Create order
	order := &domain.Order{
		OrderNumber:   orderNumber,
		CustomerEmail: req.CustomerEmail,
		CustomerName:  req.CustomerName,
		CustomerPhone: &req.CustomerPhone,
		CustomerIP:    &req.CustomerIP,
		CustomerUserAgent: &req.CustomerUserAgent,
		Currency:      req.Currency,
		Subtotal:      subtotal,
		TaxAmount:     taxAmount,
		DiscountAmount: discountAmount,
		TotalAmount:   totalAmount,
		PaymentGateway: "razorpay",
		Status:        domain.OrderStatusPending,
		IdempotencyKey: &req.IdempotencyKey,
		Metadata:      make(domain.JSONMap),
	}

	// Add billing address if provided
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

	// 6. Save order
	if err := s.orderRepo.Create(ctx, order); err != nil {
		return nil, err
	}

	// 7. Save order items
	for _, item := range orderItems {
		item.OrderID = order.ID
	}
	if err := s.orderItemRepo.CreateBatch(ctx, orderItems); err != nil {
		return nil, err
	}

	// 8. Reserve stock
	for _, item := range orderItems {
		if err := s.templateRepo.DecrementStock(ctx, item.TemplateID, item.Quantity); err != nil {
			logger.Error("Failed to reserve stock", 
				zap.Int64("template_id", item.TemplateID),
				zap.Error(err),
			)
			// Continue - stock reservation is best-effort
		}
	}

	// 9. Save idempotency key
	expiresAt := time.Now().Add(24 * time.Hour)
	idempotencyKey := &domain.IdempotencyKey{
		Key:           req.IdempotencyKey,
		OperationType: "create_order",
		EntityType:    strPtr("order"),
		EntityID:      &order.ID,
		ExpiresAt:     expiresAt,
	}
	_ = s.idempotencyRepo.Create(ctx, idempotencyKey)

	// 10. Log activity
	s.logActivity(ctx, "order_created", order.ID, 0, map[string]interface{}{
		"order_number": order.OrderNumber,
		"total_amount": order.TotalAmount,
		"currency":     order.Currency,
	})

	logger.Info("Order created successfully",
		zap.String("order_number", order.OrderNumber),
		zap.Float64("total", order.TotalAmount),
		zap.String("currency", order.Currency),
	)

	return order, nil
}

// ============================================================================
// INITIATE PAYMENT - Create Razorpay order
// ============================================================================

func (s *OrderService) InitiatePayment(ctx context.Context, orderID int64) (*domain.Payment, error) {
	// 1. Get order
	order, err := s.orderRepo.FindByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	// 2. Validate state
	if order.Status != domain.OrderStatusPending {
		return nil, fmt.Errorf("order must be in pending state, current: %s", order.Status)
	}

	// 3. Create Razorpay order
	razorpayOrder, err := s.paymentService.CreateOrder(ctx, &CreateRazorpayOrderRequest{
		Amount:   order.TotalAmount,
		Currency: order.Currency,
		Receipt:  order.OrderNumber,
		Notes: map[string]string{
			"order_id":     fmt.Sprintf("%d", order.ID),
			"order_number": order.OrderNumber,
			"customer_email": order.CustomerEmail,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create razorpay order: %w", err)
	}

	// 4. Update order
	order.GatewayOrderID = &razorpayOrder.ID
	order.Status = domain.OrderStatusPaymentInitiated
	if err := s.orderRepo.Update(ctx, order); err != nil {
		return nil, err
	}

	// 5. Create payment record
	payment := &domain.Payment{
		OrderID:        order.ID,
		Gateway:        "razorpay",
		GatewayOrderID: razorpayOrder.ID,
		Amount:         order.TotalAmount,
		Currency:       order.Currency,
		Status:         domain.PaymentStatusCreated,
		GatewayResponse: domain.JSONMap{
			"order_id": razorpayOrder.ID,
			"amount":   razorpayOrder.Amount,
			"currency": razorpayOrder.Currency,
		},
	}

	if err := s.paymentRepo.Create(ctx, payment); err != nil {
		return nil, err
	}

	// 6. Log activity
	s.logActivity(ctx, "payment_initiated", order.ID, 0, map[string]interface{}{
		"gateway_order_id": razorpayOrder.ID,
		"amount":           order.TotalAmount,
	})

	logger.Info("Payment initiated",
		zap.String("order_number", order.OrderNumber),
		zap.String("gateway_order_id", razorpayOrder.ID),
	)

	return payment, nil
}

// ============================================================================
// VERIFY PAYMENT - Signature verification + idempotency
// ============================================================================

type VerifyPaymentRequest struct {
	OrderID          int64  `json:"order_id" validate:"required"`
	RazorpayOrderID  string `json:"razorpay_order_id" validate:"required"`
	RazorpayPaymentID string `json:"razorpay_payment_id" validate:"required"`
	RazorpaySignature string `json:"razorpay_signature" validate:"required"`
	IdempotencyKey   string `json:"idempotency_key" validate:"required"`
}

func (s *OrderService) VerifyPayment(ctx context.Context, req *VerifyPaymentRequest) (*domain.Order, error) {
	// 1. Check idempotency
	existingKey, err := s.idempotencyRepo.FindByKey(ctx, req.IdempotencyKey)
	if err == nil {
		logger.Info("Duplicate payment verification detected", zap.String("key", req.IdempotencyKey))
		
		if existingKey.EntityID != nil {
			order, err := s.orderRepo.FindByID(ctx, *existingKey.EntityID)
			if err == nil {
				return order, nil
			}
		}
	}

	// 2. Get order
	order, err := s.orderRepo.FindByID(ctx, req.OrderID)
	if err != nil {
		return nil, err
	}

	// 3. Validate order state
	if order.Status != domain.OrderStatusPaymentInitiated && 
	   order.Status != domain.OrderStatusPaymentProcessing {
		return nil, fmt.Errorf("invalid order state for payment verification: %s", order.Status)
	}

	// 4. Get payment record
	payment, err := s.paymentRepo.FindByGatewayOrderID(ctx, req.RazorpayOrderID)
	if err != nil {
		return nil, fmt.Errorf("payment record not found")
	}

	// 5. CRITICAL: Verify signature
	isValid := s.paymentService.VerifyPaymentSignature(
		req.RazorpayOrderID,
		req.RazorpayPaymentID,
		req.RazorpaySignature,
	)

	if !isValid {
		// Mark as failed
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

	// 6. Update payment record
	payment.GatewayPaymentID = &req.RazorpayPaymentID
	payment.GatewaySignature = &req.RazorpaySignature
	payment.SignatureVerified = true
	payment.Status = domain.PaymentStatusCaptured
	now := time.Now()
	payment.VerifiedAt = &now
	payment.CapturedAt = &now

	if err := s.paymentRepo.Update(ctx, payment); err != nil {
		return nil, err
	}

	// 7. Update order to PAID
	order.Status = domain.OrderStatusPaid
	order.GatewayPaymentID = &req.RazorpayPaymentID
	order.GatewaySignature = &req.RazorpaySignature
	order.PaidAt = &now

	// Auto-approve or send to admin review based on amount
	if order.TotalAmount < 10000 { // Auto-approve orders under ₹10,000
		order.Status = domain.OrderStatusApproved
		order.ApprovedAt = &now
		order.DownloadsEnabled = true
		expiresAt := now.AddDate(0, 0, 30)
		order.DownloadsExpiresAt = &expiresAt
	} else {
		order.Status = domain.OrderStatusAdminReview
	}

	if err := s.orderRepo.Update(ctx, order); err != nil {
		return nil, err
	}

	// 8. Save idempotency key
	expiresAt := time.Now().Add(24 * time.Hour)
	idempotencyKey := &domain.IdempotencyKey{
		Key:           req.IdempotencyKey,
		OperationType: "verify_payment",
		EntityType:    strPtr("order"),
		EntityID:      &order.ID,
		ExpiresAt:     expiresAt,
	}
	_ = s.idempotencyRepo.Create(ctx, idempotencyKey)

	// 9. Enqueue background jobs
	if order.Status == domain.OrderStatusApproved {
		// Generate download tokens
		_ = s.enqueueJob(ctx, "generate_download_tokens", map[string]interface{}{
			"order_id": order.ID,
		})

		// Send confirmation email
		_ = s.enqueueJob(ctx, "send_order_confirmation_email", map[string]interface{}{
			"order_id": order.ID,
		})
	} else {
		// Notify admin
		_ = s.enqueueJob(ctx, "send_admin_review_notification", map[string]interface{}{
			"order_id": order.ID,
		})
	}

	// 10. Log activity
	s.logActivity(ctx, "payment_verified", order.ID, 0, map[string]interface{}{
		"payment_id": payment.ID,
		"amount":     payment.Amount,
		"status":     order.Status,
	})

	logger.Info("Payment verified successfully",
		zap.String("order_number", order.OrderNumber),
		zap.String("payment_id", req.RazorpayPaymentID),
		zap.String("final_status", string(order.Status)),
	)

	return order, nil
}

// ============================================================================
// ADMIN ACTIONS - Approve/Reject
// ============================================================================

func (s *OrderService) ApproveOrder(ctx context.Context, orderID int64, adminID int64, notes *string) error {
	// 1. Approve order
	if err := s.orderRepo.Approve(ctx, orderID, adminID, notes); err != nil {
		return err
	}

	// 2. Get updated order
	order, err := s.orderRepo.FindByID(ctx, orderID)
	if err != nil {
		return err
	}

	// 3. Enqueue jobs
	_ = s.enqueueJob(ctx, "generate_download_tokens", map[string]interface{}{
		"order_id": order.ID,
	})

	_ = s.enqueueJob(ctx, "send_order_confirmation_email", map[string]interface{}{
		"order_id": order.ID,
	})

	// 4. Log activity
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
	// 1. Reject order
	if err := s.orderRepo.Reject(ctx, orderID, adminID, reason); err != nil {
		return err
	}

	// 2. Get order
	order, err := s.orderRepo.FindByID(ctx, orderID)
	if err != nil {
		return err
	}

	// 3. Enqueue refund job
	_ = s.enqueueJob(ctx, "process_refund", map[string]interface{}{
		"order_id": order.ID,
		"reason":   reason,
	})

	// 4. Notify customer
	_ = s.enqueueJob(ctx, "send_order_rejection_email", map[string]interface{}{
		"order_id": order.ID,
		"reason":   reason,
	})

	// 5. Log activity
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
	offset := (page - 1) * limit
	return s.orderRepo.FindByEmail(ctx, email, limit, offset)
}

func (s *OrderService) GetAllOrders(ctx context.Context, filters map[string]interface{}, page, limit int) ([]*domain.Order, int, error) {
	offset := (page - 1) * limit
	return s.orderRepo.GetAll(ctx, filters, limit, offset)
}

func (s *OrderService) GetOrderTransitions(ctx context.Context, orderID int64) ([]*domain.OrderStateTransition, error) {
	return s.transitionRepo.GetByOrderID(ctx, orderID)
}

// ============================================================================
// HELPER METHODS
// ============================================================================

func (s *OrderService) generateOrderNumber() string {
	// Use database function for uniqueness guarantee
	// This is a fallback - normally DB generates it
	timestamp := time.Now().Format("20060102")
	return fmt.Sprintf("ORD-%s-%s", timestamp, generateRandomString(6))
}

func (s *OrderService) enqueueJob(ctx context.Context, jobType string, payload map[string]interface{}) error {
	job := &domain.BackgroundJob{
		JobType:     jobType,
		Payload:     payload,
		Status:      domain.JobStatusPending,
		MaxRetries:  3,
		ScheduledAt: time.Now(),
		Priority:    0,
	}

	return s.jobRepo.Create(ctx, job)
}

func (s *OrderService) logActivity(ctx context.Context, action string, entityID int64, adminID int64, metadata map[string]interface{}) {
	if s.activityLogRepo == nil {
		return
	}

	// Convert metadata to JSONMap
	jsonMetadata := make(domain.JSONMap)
	for k, v := range metadata {
		jsonMetadata[k] = v
	}

	entityType := "order"
	var adminIDPtr *int64
	if adminID > 0 {
		adminIDPtr = &adminID
	}

	activity := &domain.ActivityLog{
		Action:     action,
		EntityType: &entityType,
		EntityID:   &entityID,
		AdminID:    adminIDPtr,
		Details:    jsonMetadata,
	}

	_ = s.activityLogRepo.Create(ctx, activity)
}

func generateRandomString(length int) string {
	// Use crypto/rand for production
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[i%len(charset)]
	}
	return string(b)
}