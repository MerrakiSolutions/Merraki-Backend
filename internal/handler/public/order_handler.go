package public

import (
	"github.com/gofiber/fiber/v2"
	"github.com/merraki/merraki-backend/internal/domain"
	"github.com/merraki/merraki-backend/internal/pkg/response"
	"github.com/merraki/merraki-backend/internal/pkg/validator"
	"github.com/merraki/merraki-backend/internal/service"
)

type OrderHandler struct {
	orderService *service.OrderService
}

func NewOrderHandler(orderService *service.OrderService) *OrderHandler {
	return &OrderHandler{orderService: orderService}
}

type CreateOrderRequest struct {
	TemplateIDs   []int64 `json:"template_ids" validate:"required,min=1"`
	CustomerEmail string  `json:"customer_email" validate:"required,email"`
	CustomerName  string  `json:"customer_name" validate:"required"`
	CustomerPhone string  `json:"customer_phone"`
	CurrencyCode  string  `json:"currency_code" validate:"required"`
}

func (h *OrderHandler) Create(c *fiber.Ctx) error {
	var req CreateOrderRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, err)
	}

	if err := validator.Validate(req); err != nil {
		return response.ValidationError(c, validator.FormatValidationErrors(err))
	}

	orderReq := &service.CreateOrderRequest{
		TemplateIDs:   req.TemplateIDs,
		CustomerEmail: req.CustomerEmail,
		CustomerName:  req.CustomerName,
		CustomerPhone: req.CustomerPhone,
		CurrencyCode:  req.CurrencyCode,
		IPAddress:     c.IP(),
		UserAgent:     c.Get("User-Agent"),
	}

	orderResp, err := h.orderService.CreateOrder(c.Context(), orderReq)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Created(c, "Order created successfully", fiber.Map{
		"order":             orderResp.Order,
		"items":             orderResp.Items,
		"razorpay_order_id": orderResp.RazorpayOrderID,
		"razorpay_key_id":   orderResp.RazorpayKeyID,
	})
}

type VerifyPaymentRequest struct {
	RazorpayOrderID   string `json:"razorpay_order_id" validate:"required"`
	RazorpayPaymentID string `json:"razorpay_payment_id" validate:"required"`
	RazorpaySignature string `json:"razorpay_signature" validate:"required"`
}

func (h *OrderHandler) VerifyPayment(c *fiber.Ctx) error {
	var req VerifyPaymentRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, err)
	}

	if err := validator.Validate(req); err != nil {
		return response.ValidationError(c, validator.FormatValidationErrors(err))
	}

	order, err := h.orderService.VerifyPayment(c.Context(), req.RazorpayOrderID, req.RazorpayPaymentID, req.RazorpaySignature)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, "Payment verified successfully", fiber.Map{
		"order_number": order.OrderNumber,
		"status":       order.Status,
		"email":        order.CustomerEmail,
		"next_step":    "awaiting_approval",
		"message":      "You will receive download links via email once approved",
	})
}

func (h *OrderHandler) Lookup(c *fiber.Ctx) error {
	orderNumber := c.Query("order_number")
	email := c.Query("email")

	if orderNumber == "" || email == "" {
		return response.Error(c, fiber.NewError(400, "Order number and email required"))
	}

	order, items, err := h.orderService.GetOrderByNumberAndEmail(c.Context(), orderNumber, email)
	if err != nil {
		return response.Error(c, err)
	}

	return response.SuccessData(c, fiber.Map{
		"order": order,
		"items": items,
	})
}

func (h *OrderHandler) Download(c *fiber.Ctx) error {
	orderNumber := c.Params("orderNumber")
	email := c.Query("email")
	token := c.Query("token")

	order, items, err := h.orderService.ValidateDownload(c.Context(), orderNumber, email, token)
	if err != nil {
		return response.Error(c, err)
	}

	// TODO: Generate ZIP file with all templates
	// For now, return file URLs
	
	_ = h.orderService.RecordDownload(c.Context(), order.ID, &domain.DownloadLog{
		OrderID:   order.ID,
		Status:    "success",
		IPAddress: strPtr(c.IP()),
		UserAgent: strPtr(c.Get("User-Agent")),
	})

	return response.SuccessData(c, fiber.Map{
		"order":  order,
		"items":  items,
		"message": "Download ready",
	})
}

// Add these methods to the existing order_handler.go

func (h *OrderHandler) Webhook(c *fiber.Ctx) error {
	// Verify Razorpay webhook signature
	signature := c.Get("X-Razorpay-Signature")
	if signature == "" {
		return response.Error(c, fiber.NewError(400, "Missing signature"))
	}

	_ = c.Body()
	
	// TODO: Verify webhook signature and process event
	// For now, just return success
	
	return c.JSON(fiber.Map{
		"success": true,
	})
}

func strPtr(s string) *string {
	return &s
}