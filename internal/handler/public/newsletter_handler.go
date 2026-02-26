package public

import (
	"github.com/gofiber/fiber/v2"
	"github.com/merraki/merraki-backend/internal/pkg/response"
	"github.com/merraki/merraki-backend/internal/pkg/validator"
	"github.com/merraki/merraki-backend/internal/service"
)

type NewsletterHandler struct {
	newsletterService *service.NewsletterService
}

func NewNewsletterHandler(newsletterService *service.NewsletterService) *NewsletterHandler {
	return &NewsletterHandler{newsletterService: newsletterService}
}

type SubscribeRequest struct {
	Email string `json:"email" validate:"required,email"`
	Name  string `json:"name"`
}

func (h *NewsletterHandler) Subscribe(c *fiber.Ctx) error {
	var req SubscribeRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, err)
	}

	if err := validator.Validate(req); err != nil {
		return response.ValidationError(c, validator.FormatValidationErrors(err))
	}

	subscribeReq := &service.SubscribeRequest{
		Email:     req.Email,
		Name:      req.Name,
		Source:    "website",
		IPAddress: c.IP(),
	}

	if err := h.newsletterService.Subscribe(c.Context(), subscribeReq); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, "Successfully subscribed to newsletter", nil)
}

type UnsubscribeRequest struct {
	Email string `json:"email" validate:"required,email"`
}

func (h *NewsletterHandler) Unsubscribe(c *fiber.Ctx) error {
	var req UnsubscribeRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, err)
	}

	if err := validator.Validate(req); err != nil {
		return response.ValidationError(c, validator.FormatValidationErrors(err))
	}

	if err := h.newsletterService.Unsubscribe(c.Context(), req.Email); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, "Successfully unsubscribed from newsletter", nil)
}

func (h *NewsletterHandler) UnsubscribeGET(c *fiber.Ctx) error {
	email := c.Query("email")
	if email == "" {
		return response.Error(c, fiber.NewError(400, "Email required"))
	}

	if err := h.newsletterService.Unsubscribe(c.Context(), email); err != nil {
		return response.Error(c, err)
	}

	return c.SendString("You have been unsubscribed from our newsletter.")
}