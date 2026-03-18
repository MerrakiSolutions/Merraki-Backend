package public

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/merraki/merraki-backend/internal/service"
)

// ============================================================================
// ORDER HANDLER - Order lookup & tracking
// ============================================================================

type OrderHandler struct {
	orderService *service.OrderService
}

func NewOrderHandler(orderService *service.OrderService) *OrderHandler {
	return &OrderHandler{
		orderService: orderService,
	}
}

// ============================================================================
// LOOKUP ORDER BY NUMBER
// ============================================================================

// GET /api/v1/orders/lookup?order_number=xxx&email=xxx
func (h *OrderHandler) LookupOrder(c *fiber.Ctx) error {
	orderNumber := c.Query("order_number")
	email := c.Query("email")

	if orderNumber == "" || email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Order number and email are required",
		})
	}

	// Get order
	order, err := h.orderService.GetOrderByNumber(c.Context(), orderNumber)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Order not found",
		})
	}

	// Verify email matches
	if order.Order.CustomerEmail != email {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	return c.JSON(fiber.Map{
		"order": order,
	})
}

// ============================================================================
// GET ORDER BY ID (with email verification)
// ============================================================================

// GET /api/v1/orders/:id?email=xxx
func (h *OrderHandler) GetOrderByID(c *fiber.Ctx) error {
	idStr := c.Params("id")
	email := c.Query("email")

	if email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Email is required",
		})
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid order ID",
		})
	}

	// Get order
	order, err := h.orderService.GetOrderByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Order not found",
		})
	}

	// Verify email matches
	if order.Order.CustomerEmail != email {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	return c.JSON(fiber.Map{
		"order": order,
	})
}

// ============================================================================
// GET ORDERS BY EMAIL
// ============================================================================

// GET /api/v1/orders/by-email?email=xxx&page=1&limit=10
func (h *OrderHandler) GetOrdersByEmail(c *fiber.Ctx) error {
	email := c.Query("email")
	if email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Email is required",
		})
	}

	// Parse pagination
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	orders, total, err := h.orderService.GetOrdersByEmail(
		c.Context(),
		email,
		page,
		limit,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get orders",
		})
	}

	return c.JSON(fiber.Map{
		"orders": orders,
		"total":  total,
		"page":   page,
		"limit":  limit,
	})
}