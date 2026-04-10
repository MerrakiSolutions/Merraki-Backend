package public

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/merraki/merraki-backend/internal/domain"
	"github.com/merraki/merraki-backend/internal/pkg/logger"
	"github.com/merraki/merraki-backend/internal/service"
	"go.uber.org/zap"
)

// ----------------------------------------------------------------------
// Validator singleton
// ----------------------------------------------------------------------

var validate = validator.New()

// ----------------------------------------------------------------------
// Checkout Handler
// ----------------------------------------------------------------------

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

// ----------------------------------------------------------------------
// CREATE ORDER
// ----------------------------------------------------------------------

type CreateOrderRequest struct {
	CustomerEmail  string                    `json:"customer_email"  validate:"required,email"`
	CustomerName   string                    `json:"customer_name"   validate:"required"`
	CustomerPhone  string                    `json:"customer_phone"`
	BillingAddress *domain.BillingAddress   `json:"billing_address"`
	Items          []service.CreateOrderItem `json:"items" validate:"required,min=1,dive"`

	// ✅ frontend-generated idempotency key
	IdempotencyKey string `json:"idempotency_key" validate:"required"`
}

func (h *CheckoutHandler) CreateOrder(c *fiber.Ctx) error {
	var req CreateOrderRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if err := validate.Struct(&req); err != nil {
		return c.Status(422).JSON(fiber.Map{
			"error":  "validation failed",
			"detail": err.Error(),
		})
	}

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

	order, err := h.orderService.CreateOrder(c.Context(), serviceReq)
	if err != nil {
		logger.Error("create order failed", zap.Error(err))
		return c.Status(422).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(201).JSON(fiber.Map{"order": order})
}

// ----------------------------------------------------------------------
// INITIATE PAYMENT
// ----------------------------------------------------------------------

type InitiatePaymentRequest struct {
	OrderID int64 `json:"order_id" validate:"required"`
}

func (h *CheckoutHandler) InitiatePayment(c *fiber.Ctx) error {
	var req InitiatePaymentRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if err := validate.Struct(&req); err != nil {
		return c.Status(422).JSON(fiber.Map{"error": err.Error()})
	}

	payment, err := h.orderService.InitiatePayment(c.Context(), req.OrderID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return c.Status(404).JSON(fiber.Map{"error": "order not found"})
		}
		return c.Status(409).JSON(fiber.Map{"error": err.Error()})
	}

	order, err := h.orderService.GetOrderByID(c.Context(), req.OrderID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to fetch order"})
	}

	return c.JSON(fiber.Map{
		"razorpay_order_id": payment.GatewayOrderID,
		"amount_cents":      payment.AmountUSDCents,
		"key_id":            h.paymentService.GetKeyID(),
		"order_number":      order.Order.OrderNumber,
	})
}

// ----------------------------------------------------------------------
// VERIFY PAYMENT
// ----------------------------------------------------------------------

type VerifyPaymentRequest struct {
	OrderID           int64  `json:"order_id" validate:"required"`
	RazorpayOrderID   string `json:"razorpay_order_id" validate:"required"`
	RazorpayPaymentID string `json:"razorpay_payment_id" validate:"required"`
	RazorpaySignature string `json:"razorpay_signature" validate:"required"`
	IdempotencyKey    string `json:"idempotency_key" validate:"required"`
}

func (h *CheckoutHandler) VerifyPayment(c *fiber.Ctx) error {
	var req VerifyPaymentRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if err := validate.Struct(&req); err != nil {
		return c.Status(422).JSON(fiber.Map{"error": err.Error()})
	}

	serviceReq := &service.VerifyPaymentRequest{
		OrderID:           req.OrderID,
		RazorpayOrderID:   req.RazorpayOrderID,
		RazorpayPaymentID: req.RazorpayPaymentID,
		RazorpaySignature: req.RazorpaySignature,
		IdempotencyKey:    req.IdempotencyKey,
	}

	order, err := h.orderService.VerifyPayment(c.Context(), serviceReq)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return c.Status(404).JSON(fiber.Map{"error": "order not found"})
		}
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"order":   order,
		"status":  order.Status,
	})
}

// ----------------------------------------------------------------------
// WEBHOOK
// ----------------------------------------------------------------------

func (h *CheckoutHandler) HandleWebhook(c *fiber.Ctx) error {
	payload := c.Body()

	if len(payload) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "empty payload"})
	}

	signature := c.Get("X-Razorpay-Signature")
	if signature == "" {
		return c.Status(400).JSON(fiber.Map{"error": "missing signature"})
	}

	result, err := h.paymentService.ProcessWebhook(
		c.Context(),
		payload,
		signature,
		c.IP(),
		string(c.Request().Header.UserAgent()),
	)

	if err != nil {
		logger.Error("webhook failed", zap.Error(err))

		if err.Error() == "invalid webhook signature" {
			return c.JSON(fiber.Map{"status": "received"})
		}

		return c.Status(500).JSON(fiber.Map{"error": "webhook processing failed"})
	}

	// ------------------------------------------------------------------
	// EVENT ROUTING (business layer)
	// ------------------------------------------------------------------

	switch result.Event {

	case "payment.captured":
		err := h.orderService.MarkPaymentCaptured(
			c.Context(),
			result.GatewayOrderID,
			result.GatewayPaymentID,
		)
		if err != nil {
			logger.Error("mark captured failed", zap.Error(err))
		}

	case "payment.failed":
		err := h.orderService.MarkPaymentFailed(
			c.Context(),
			result.GatewayOrderID,
		)
		if err != nil {
			logger.Error("mark failed error", zap.Error(err))
		}

	default:
		logger.Warn("unhandled webhook event", zap.String("event", result.Event))
	}

	return c.JSON(fiber.Map{"status": "received"})
}

// ----------------------------------------------------------------------

func validateStruct(s interface{}) error {
	return validate.Struct(s)
}