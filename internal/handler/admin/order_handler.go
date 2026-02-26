package admin

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/merraki/merraki-backend/internal/domain"
	"github.com/merraki/merraki-backend/internal/middleware"
	"github.com/merraki/merraki-backend/internal/pkg/response"
	"github.com/merraki/merraki-backend/internal/service"
)

type OrderHandler struct {
	orderService *service.OrderService
}

func NewOrderHandler(orderService *service.OrderService) *OrderHandler {
	return &OrderHandler{orderService: orderService}
}

func (h *OrderHandler) GetAll(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	params := &domain.PaginationParams{
		Page:  page,
		Limit: limit,
	}
	params.Validate()

	filters := make(map[string]interface{})
	if status := c.Query("status"); status != "" {
		filters["status"] = status
	}
	if paymentStatus := c.Query("payment_status"); paymentStatus != "" {
		filters["payment_status"] = paymentStatus
	}
	if search := c.Query("search"); search != "" {
		filters["search"] = search
	}
	if startDate := c.Query("start_date"); startDate != "" {
		filters["start_date"] = startDate
	}
	if endDate := c.Query("end_date"); endDate != "" {
		filters["end_date"] = endDate
	}
	if sort := c.Query("sort"); sort != "" {
		filters["sort"] = sort
	}

	orders, total, err := h.orderService.GetAllOrders(c.Context(), filters, params.Limit, params.GetOffset())
	if err != nil {
		return response.Error(c, err)
	}

	return response.Paginated(c, orders, total, params.Page, params.Limit)
}

func (h *OrderHandler) GetPending(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))

	params := &domain.PaginationParams{
		Page:  page,
		Limit: limit,
	}
	params.Validate()

	orders, total, err := h.orderService.GetPendingApprovals(c.Context(), params.Limit, params.GetOffset())
	if err != nil {
		return response.Error(c, err)
	}

	return response.Paginated(c, orders, total, params.Page, params.Limit)
}

func (h *OrderHandler) GetByID(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, fiber.NewError(400, "Invalid order ID"))
	}

	order, items, err := h.orderService.GetOrderByID(c.Context(), int64(id))
	if err != nil {
		return response.Error(c, err)
	}

	return response.SuccessData(c, fiber.Map{
		"order": order,
		"items": items,
	})
}

func (h *OrderHandler) Approve(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, fiber.NewError(400, "Invalid order ID"))
	}

	adminID := middleware.GetAdminID(c)

	if err := h.orderService.ApproveOrder(c.Context(), int64(id), adminID); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, "Order approved and download link sent to customer", nil)
}

type RejectOrderRequest struct {
	Reason string `json:"reason" validate:"required"`
}

func (h *OrderHandler) Reject(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, fiber.NewError(400, "Invalid order ID"))
	}

	var req RejectOrderRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, err)
	}

	adminID := middleware.GetAdminID(c)

	if err := h.orderService.RejectOrder(c.Context(), int64(id), adminID, req.Reason); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, "Order rejected and customer notified", nil)
}

func (h *OrderHandler) GetRevenueAnalytics(c *fiber.Ctx) error {
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	groupBy := c.Query("group_by", "day")

	analytics, err := h.orderService.GetRevenueAnalytics(c.Context(), startDate, endDate, groupBy)
	if err != nil {
		return response.Error(c, err)
	}

	return response.SuccessData(c, analytics)
}