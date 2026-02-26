package public

import (
	"github.com/gofiber/fiber/v2"
	"github.com/merraki/merraki-backend/internal/pkg/response"
	"github.com/merraki/merraki-backend/internal/pkg/validator"
	"github.com/merraki/merraki-backend/internal/service"
)

type ContactHandler struct {
	contactService *service.ContactService
}

func NewContactHandler(contactService *service.ContactService) *ContactHandler {
	return &ContactHandler{contactService: contactService}
}

type CreateContactRequest struct {
	Name    string `json:"name" validate:"required"`
	Email   string `json:"email" validate:"required,email"`
	Phone   string `json:"phone"`
	Subject string `json:"subject" validate:"required"`
	Message string `json:"message" validate:"required"`
}

func (h *ContactHandler) Create(c *fiber.Ctx) error {
	var req CreateContactRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, err)
	}

	if err := validator.Validate(req); err != nil {
		return response.ValidationError(c, validator.FormatValidationErrors(err))
	}

	contactReq := &service.CreateContactRequest{
		Name:      req.Name,
		Email:     req.Email,
		Phone:     req.Phone,
		Subject:   req.Subject,
		Message:   req.Message,
		IPAddress: c.IP(),
	}

	contact, err := h.contactService.CreateContact(c.Context(), contactReq)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Created(c, "Message sent successfully. We'll respond within 24 hours.", contact)
}