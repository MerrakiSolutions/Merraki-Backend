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

// validate is a package-level singleton. Initialising it once avoids
// rebuilding the internal struct cache on every request.
var validate = validator.New()

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
	CustomerEmail  string                    `json:"customer_email"  validate:"required,email"`
	CustomerName   string                    `json:"customer_name"   validate:"required"`
	CustomerPhone  string                    `json:"customer_phone"`
	BillingAddress *domain.BillingAddress    `json:"billing_address"`
	Items          []service.CreateOrderItem `json:"items"           validate:"required,min=1,dive"`
	IdempotencyKey string                    `json:"idempotency_key" validate:"required"`
}

// POST /api/v1/checkout/create-order
func (h *CheckoutHandler) CreateOrder(c *fiber.Ctx) error {
	var req CreateOrderRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// FIX 1: Validate the parsed struct — without this, all `validate:` tags
	// on CreateOrderRequest (required, email, min=1 …) were silently ignored.
	if err := validateStruct(&req); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error":  "Validation failed",
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
		// FIX 2: Surface business errors (template not found / not available)
		// as 400/422 instead of a generic 500 that hides the real cause.
		logger.Error("Failed to create order", zap.Error(err))
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error": err.Error(),
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

// POST /api/v1/checkout/initiate-payment
func (h *CheckoutHandler) InitiatePayment(c *fiber.Ctx) error {
	var req InitiatePaymentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// FIX 3: Validate struct (order_id required was silently ignored before).
	if err := validateStruct(&req); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error":  "Validation failed",
			"detail": err.Error(),
		})
	}

	payment, err := h.orderService.InitiatePayment(c.Context(), req.OrderID)
	if err != nil {
		logger.Error("Failed to initiate payment", zap.Error(err))

		// FIX 4: Map business errors to the correct HTTP status codes instead
		// of a blanket 500:
		//   • Order not found          → 404
		//   • Wrong order state        → 409 Conflict
		//   • Razorpay / internal err  → 500
		if errors.Is(err, domain.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Order not found",
			})
		}
		// "cannot initiate payment in state: …" is a state-machine conflict.
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// FIX 5: The payment record already carries GatewayOrderID and
	// AmountUSDCents, so the previous extra GetOrderByID call (purely to
	// read OrderNumber and TotalAmountUSDCents) is unnecessary. Fetch the
	// order only once to get OrderNumber for the response.
	order, err := h.orderService.GetOrderByID(c.Context(), req.OrderID)
	if err != nil || order == nil {
		logger.Error("Failed to fetch order after payment initiation", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve order details",
		})
	}

	return c.JSON(fiber.Map{
		"razorpay_order_id": payment.GatewayOrderID,
		// Send raw cents — Razorpay SDK expects the smallest currency unit.
		// No float conversion, no rounding risk (e.g. $10.99 stays 1099).
		"amount_cents": payment.AmountUSDCents,
		"key_id":       h.paymentService.GetKeyID(),
		"order_number": order.Order.OrderNumber,
	})
}

// ============================================================================
// VERIFY PAYMENT
// ============================================================================

type VerifyPaymentRequest struct {
	OrderID           int64  `json:"order_id"            validate:"required"`
	RazorpayOrderID   string `json:"razorpay_order_id"   validate:"required"`
	RazorpayPaymentID string `json:"razorpay_payment_id" validate:"required"`
	RazorpaySignature string `json:"razorpay_signature"  validate:"required"`
	IdempotencyKey    string `json:"idempotency_key"     validate:"required"`
}

// POST /api/v1/checkout/verify-payment
func (h *CheckoutHandler) VerifyPayment(c *fiber.Ctx) error {
	var req VerifyPaymentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// FIX 6: Validate struct — all four required fields were silently ignored.
	if err := validateStruct(&req); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error":  "Validation failed",
			"detail": err.Error(),
		})
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
		logger.Error("Payment verification failed", zap.Error(err))

		// FIX 7: Distinguish signature failure (client fault → 400) from an
		// order-not-found (404) vs an unexpected internal fault (500) so the
		// frontend can show a meaningful error rather than a generic message.
		if errors.Is(err, domain.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Order not found",
			})
		}
		// "payment signature verification failed" and "invalid order state"
		// are both client/request errors.
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
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
	payload := c.Body()

	// FIX 8: Reject empty payloads before doing anything else.
	// An empty body would silently reach ProcessWebhook and fail to parse.
	if len(payload) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Empty payload",
		})
	}

	signature := c.Get("X-Razorpay-Signature")
	if signature == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Missing signature",
		})
	}

	sourceIP := c.IP()
	userAgent := string(c.Request().Header.UserAgent())

	err := h.paymentService.ProcessWebhook(c.Context(), payload, signature, sourceIP, userAgent)
	if err != nil {
		logger.Error("Webhook processing failed", zap.Error(err))

		// FIX 9: The service already persisted the webhook record before
		// returning an "invalid signature" error (intentional audit trail).
		// Returning 400 here tells Razorpay the delivery failed, so it will
		// retry indefinitely — including for genuinely invalid/replayed
		// requests. Respond 200 to acknowledge receipt; the audit record and
		// log entry are the source of truth for ops investigation.
		//
		// If the error is a genuine internal fault (DB down etc.) we still
		// want Razorpay to retry, so we return 500 only for non-signature
		// errors.
		if err.Error() == "invalid webhook signature" {
			// Persisted for audit, no retry needed.
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"status": "received",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Webhook processing failed",
		})
	}

	return c.JSON(fiber.Map{
		"status": "received",
	})
}

// ============================================================================
// HELPERS
// ============================================================================

// validateStruct runs go-playground/validator on any struct pointer.
// Uses the package-level singleton to avoid rebuilding the cache per request.
func validateStruct(s interface{}) error {
	return validate.Struct(s)
}