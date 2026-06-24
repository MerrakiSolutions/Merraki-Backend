package public

import (
	"github.com/gofiber/fiber/v2"
	"github.com/merraki/merraki-backend/internal/domain"
	"github.com/merraki/merraki-backend/internal/pkg/logger"
	"github.com/merraki/merraki-backend/internal/service"
	"go.uber.org/zap"
)

// ============================================================================
// PUBLIC FOUNDER LEAD HANDLER
// ============================================================================

type FounderLeadHandler struct {
	service *service.FounderLeadService
}

func NewFounderLeadHandler(service *service.FounderLeadService) *FounderLeadHandler {
	return &FounderLeadHandler{service: service}
}

// ============================================================================
// SUBMIT - POST /api/v1/public/founders-test/submit
// ============================================================================

func (h *FounderLeadHandler) Submit(c *fiber.Ctx) error {
	var req domain.SubmitFounderLeadRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	// Get client IP
	ip := c.IP()

	// Submit lead
	lead, err := h.service.SubmitLead(c.Context(), &req, ip)
	if err != nil {
		logger.Error("Failed to submit founder lead", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to save test result",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    lead,
	})
}
