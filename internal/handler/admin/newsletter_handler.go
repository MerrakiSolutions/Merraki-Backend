package admin

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/merraki/merraki-backend/internal/domain"
	"github.com/merraki/merraki-backend/internal/pkg/response"
	"github.com/merraki/merraki-backend/internal/service"
)

type NewsletterHandler struct {
	newsletterService *service.NewsletterService
}

func NewNewsletterHandler(newsletterService *service.NewsletterService) *NewsletterHandler {
	return &NewsletterHandler{newsletterService: newsletterService}
}

func (h *NewsletterHandler) GetAll(c *fiber.Ctx) error {
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
	if search := c.Query("search"); search != "" {
		filters["search"] = search
	}

	subscribers, total, err := h.newsletterService.GetAllSubscribers(c.Context(), filters, params.Limit, params.GetOffset())
	if err != nil {
		return response.Error(c, err)
	}

	return response.Paginated(c, subscribers, total, params.Page, params.Limit)
}

func (h *NewsletterHandler) Delete(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, fiber.NewError(400, "Invalid subscriber ID"))
	}

	if err := h.newsletterService.DeleteSubscriber(c.Context(), int64(id)); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, "Subscriber deleted successfully", nil)
}

func (h *NewsletterHandler) GetAnalytics(c *fiber.Ctx) error {
	analytics, err := h.newsletterService.GetAnalytics(c.Context())
	if err != nil {
		return response.Error(c, err)
	}

	return response.SuccessData(c, analytics)
}

// Add these methods to existing newsletter_handler.go

func (h *NewsletterHandler) Add(c *fiber.Ctx) error {
	var req struct {
		Email string `json:"email" validate:"required,email"`
		Name  string `json:"name"`
	}

	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, err)
	}

	subscribeReq := &service.SubscribeRequest{
		Email:     req.Email,
		Name:      req.Name,
		Source:    "admin_import",
		IPAddress: c.IP(),
	}

	if err := h.newsletterService.Subscribe(c.Context(), subscribeReq); err != nil {
		return response.Error(c, err)
	}

	return response.Created(c, "Subscriber added successfully", nil)
}

func (h *NewsletterHandler) Export(c *fiber.Ctx) error {
	// TODO: Export subscribers to CSV
	return response.Error(c, fiber.NewError(501, "Export not yet implemented"))
}