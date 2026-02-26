package admin

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/merraki/merraki-backend/internal/domain"
	"github.com/merraki/merraki-backend/internal/middleware"
	"github.com/merraki/merraki-backend/internal/pkg/response"
	"github.com/merraki/merraki-backend/internal/service"
)

type ContactHandler struct {
	contactService *service.ContactService
}

func NewContactHandler(contactService *service.ContactService) *ContactHandler {
	return &ContactHandler{contactService: contactService}
}

func (h *ContactHandler) GetAll(c *fiber.Ctx) error {
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

	contacts, total, err := h.contactService.GetAllContacts(c.Context(), filters, params.Limit, params.GetOffset())
	if err != nil {
		return response.Error(c, err)
	}

	return response.Paginated(c, contacts, total, params.Page, params.Limit)
}

func (h *ContactHandler) GetByID(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, fiber.NewError(400, "Invalid contact ID"))
	}

	contact, err := h.contactService.GetContactByID(c.Context(), int64(id))
	if err != nil {
		return response.Error(c, err)
	}

	return response.SuccessData(c, contact)
}

type UpdateContactRequest struct {
	Status     string `json:"status" validate:"required,oneof=new in_progress replied closed"`
	ReplyNotes string `json:"reply_notes"`
}

func (h *ContactHandler) Update(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, fiber.NewError(400, "Invalid contact ID"))
	}

	var req UpdateContactRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, err)
	}

	adminID := middleware.GetAdminID(c)

	contact, err := h.contactService.GetContactByID(c.Context(), int64(id))
	if err != nil {
		return response.Error(c, err)
	}

	contact.Status = req.Status
	if req.ReplyNotes != "" {
		contact.ReplyNotes = &req.ReplyNotes
	}

	if err := h.contactService.UpdateContact(c.Context(), contact, adminID); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, "Contact updated successfully", contact)
}

type ReplyContactRequest struct {
	Message string `json:"message" validate:"required"`
}

func (h *ContactHandler) Reply(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, fiber.NewError(400, "Invalid contact ID"))
	}

	var req ReplyContactRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, err)
	}

	adminID := middleware.GetAdminID(c)

	if err := h.contactService.ReplyToContact(c.Context(), int64(id), req.Message, adminID); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, "Reply sent successfully", nil)
}

func (h *ContactHandler) Delete(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, fiber.NewError(400, "Invalid contact ID"))
	}

	adminID := middleware.GetAdminID(c)

	if err := h.contactService.DeleteContact(c.Context(), int64(id), adminID); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, "Contact deleted successfully", nil)
}

func (h *ContactHandler) GetAnalytics(c *fiber.Ctx) error {
	analytics, err := h.contactService.GetAnalytics(c.Context())
	if err != nil {
		return response.Error(c, err)
	}

	return response.SuccessData(c, analytics)
}