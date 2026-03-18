package admin

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/merraki/merraki-backend/internal/domain"
	"github.com/merraki/merraki-backend/internal/pkg/logger"
	"github.com/merraki/merraki-backend/internal/service"
	"go.uber.org/zap"
)

// ============================================================================
// ADMIN ORDER HANDLER - Order management
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
// GET ALL ORDERS
// ============================================================================

// GET /api/v1/admin/orders?status=pending&page=1&limit=20
func (h *OrderHandler) GetAllOrders(c *fiber.Ctx) error {
	// Parse query parameters
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// Build filters
	filters := make(map[string]interface{})
	
	if status := c.Query("status"); status != "" {
		filters["status"] = domain.OrderStatus(status)
	}

	if email := c.Query("email"); email != "" {
		filters["email"] = email
	}

	orders, total, err := h.orderService.GetAllOrders(
		c.Context(),
		filters,
		page,
		limit,
	)
	if err != nil {
		logger.Error("Failed to get orders", zap.Error(err))
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

// ============================================================================
// GET ORDER BY ID
// ============================================================================

// GET /api/v1/admin/orders/:id
func (h *OrderHandler) GetOrderByID(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid order ID",
		})
	}

	order, err := h.orderService.GetOrderByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Order not found",
		})
	}

	// Get state transitions
	transitions, _ := h.orderService.GetOrderTransitions(c.Context(), id)

	return c.JSON(fiber.Map{
		"order":       order,
		"transitions": transitions,
	})
}

// ============================================================================
// APPROVE ORDER
// ============================================================================

type ApproveOrderRequest struct {
	Notes *string `json:"notes"`
}

// POST /api/v1/admin/orders/:id/approve
func (h *OrderHandler) ApproveOrder(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid order ID",
		})
	}

	var req ApproveOrderRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Get admin ID from context (set by auth middleware)
	adminID := c.Locals("admin_id").(int64)

	err = h.orderService.ApproveOrder(c.Context(), id, adminID, req.Notes)
	if err != nil {
		logger.Error("Failed to approve order", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to approve order",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Order approved successfully",
	})
}

// ============================================================================
// REJECT ORDER
// ============================================================================

type RejectOrderRequest struct {
	Reason string `json:"reason" validate:"required"`
}

// POST /api/v1/admin/orders/:id/reject
func (h *OrderHandler) RejectOrder(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid order ID",
		})
	}

	var req RejectOrderRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Reason == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Rejection reason is required",
		})
	}

	// Get admin ID from context
	adminID := c.Locals("admin_id").(int64)

	err = h.orderService.RejectOrder(c.Context(), id, adminID, req.Reason)
	if err != nil {
		logger.Error("Failed to reject order", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to reject order",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Order rejected successfully",
	})
}

// ============================================================================
// GET ORDERS PENDING REVIEW
// ============================================================================

// GET /api/v1/admin/orders/pending-review?page=1&limit=20
func (h *OrderHandler) GetPendingReviewOrders(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	filters := map[string]interface{}{
		"status": domain.OrderStatusAdminReview,
	}

	orders, total, err := h.orderService.GetAllOrders(
		c.Context(),
		filters,
		page,
		limit,
	)
	if err != nil {
		logger.Error("Failed to get pending orders", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get pending orders",
		})
	}

	return c.JSON(fiber.Map{
		"orders": orders,
		"total":  total,
		"page":   page,
		"limit":  limit,
	})
}