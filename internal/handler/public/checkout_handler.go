package public

import (
	"github.com/gofiber/fiber/v2"
	"github.com/merraki/merraki-backend/internal/domain"
	"github.com/merraki/merraki-backend/internal/pkg/logger"
	"github.com/merraki/merraki-backend/internal/service"
	"go.uber.org/zap"
)

// ============================================================================
// CHECKOUT HANDLER - Guest checkout & payment
// ============================================================================

type CheckoutHandler struct {
	orderService   *service.OrderService
	paymentService *service.PaymentService
}

func NewCheckoutHandler(
	orderService *service.OrderService,
	paymentService *service.PaymentService,
) *CheckoutHandler {
	return &CheckoutHandler{
		orderService:   orderService,
		paymentService: paymentService,
	}
}

// ============================================================================
// CREATE ORDER
// ============================================================================

type CreateOrderRequest struct {
	CustomerEmail  string                      `json:"customer_email" validate:"required,email"`
	CustomerName   string                      `json:"customer_name" validate:"required"`
	CustomerPhone  string                      `json:"customer_phone"`
	BillingAddress *domain.BillingAddress      `json:"billing_address"`
	Items          []service.CreateOrderItem   `json:"items" validate:"required,min=1,dive"`
	IdempotencyKey string                      `json:"idempotency_key" validate:"required"`
}

// POST /api/v1/checkout/create-order
func (h *CheckoutHandler) CreateOrder(c *fiber.Ctx) error {
	var req CreateOrderRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Build service request
	serviceReq := &service.CreateOrderRequest{
		CustomerEmail:     req.CustomerEmail,
		CustomerName:      req.CustomerName,
		CustomerPhone:     req.CustomerPhone,
		BillingAddress:    req.BillingAddress,
		Items:             req.Items,
		IdempotencyKey:    req.IdempotencyKey,
		CustomerIP:        c.IP(),
		CustomerUserAgent: string(c.Request().Header.UserAgent()),
	}

	// Create order
	order, err := h.orderService.CreateOrder(c.Context(), serviceReq)
	if err != nil {
		logger.Error("Failed to create order", zap.Error(err))
		
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create order",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"order": order,
	})
}

// ============================================================================
// INITIATE PAYMENT
// ============================================================================

type InitiatePaymentRequest struct {
	OrderID int64 `json:"order_id" validate:"required"`
}

type InitiatePaymentResponse struct {
	RazorpayOrderID string  `json:"razorpay_order_id"`
	Amount          float64 `json:"amount"`
	KeyID           string  `json:"key_id"`
	OrderNumber     string  `json:"order_number"`
}

// POST /api/v1/checkout/initiate-payment
func (h *CheckoutHandler) InitiatePayment(c *fiber.Ctx) error {
	var req InitiatePaymentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Initiate payment
	payment, err := h.orderService.InitiatePayment(c.Context(), req.OrderID)
	if err != nil {
		logger.Error("Failed to initiate payment", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to initiate payment",
		})
	}

	// Get order for response
	order, err := h.orderService.GetOrderByID(c.Context(), req.OrderID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get order",
		})
	}

	return c.JSON(fiber.Map{
		"razorpay_order_id": payment.GatewayOrderID,
		"amount": domain.CentsToUSD(order.Order.TotalAmountUSDCents),
		"key_id":            h.getKeyID(),
		"order_number":      order.Order.OrderNumber,
	})
}

// ============================================================================
// VERIFY PAYMENT
// ============================================================================

type VerifyPaymentRequest struct {
	OrderID           int64  `json:"order_id" validate:"required"`
	RazorpayOrderID   string `json:"razorpay_order_id" validate:"required"`
	RazorpayPaymentID string `json:"razorpay_payment_id" validate:"required"`
	RazorpaySignature string `json:"razorpay_signature" validate:"required"`
	IdempotencyKey    string `json:"idempotency_key" validate:"required"`
}

// POST /api/v1/checkout/verify-payment
func (h *CheckoutHandler) VerifyPayment(c *fiber.Ctx) error {
	var req VerifyPaymentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Build service request
	serviceReq := &service.VerifyPaymentRequest{
		OrderID:           req.OrderID,
		RazorpayOrderID:   req.RazorpayOrderID,
		RazorpayPaymentID: req.RazorpayPaymentID,
		RazorpaySignature: req.RazorpaySignature,
		IdempotencyKey:    req.IdempotencyKey,
	}

	// Verify payment
	order, err := h.orderService.VerifyPayment(c.Context(), serviceReq)
	if err != nil {
		logger.Error("Payment verification failed", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Payment verification failed",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"order":   order,
		"status":  order.Status,
		"message": "Payment verified successfully",
	})
}

// ============================================================================
// PAYMENT WEBHOOK (Razorpay callback)
// ============================================================================

// POST /api/v1/webhooks/razorpay
func (h *CheckoutHandler) HandleWebhook(c *fiber.Ctx) error {
	// Get raw body
	payload := c.Body()

	// Get signature from header
	signature := c.Get("X-Razorpay-Signature")
	if signature == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Missing signature",
		})
	}

	// Process webhook
	sourceIP := c.IP()
	userAgent := string(c.Request().Header.UserAgent())

	err := h.paymentService.ProcessWebhook(c.Context(), payload, signature, sourceIP, userAgent)
	if err != nil {
		logger.Error("Webhook processing failed", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Webhook processing failed",
		})
	}

	return c.JSON(fiber.Map{
		"status": "received",
	})
}

// ============================================================================
// HELPER METHODS
// ============================================================================

func (h *CheckoutHandler) getKeyID() string {
	return h.paymentService.GetKeyID()
}