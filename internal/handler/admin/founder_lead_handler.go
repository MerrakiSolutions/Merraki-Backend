package admin

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/merraki/merraki-backend/internal/domain"
	"github.com/merraki/merraki-backend/internal/pkg/logger"
	"github.com/merraki/merraki-backend/internal/repository/postgres"
	"github.com/merraki/merraki-backend/internal/service"
	"go.uber.org/zap"
)

// ============================================================================
// ADMIN FOUNDER LEAD HANDLER
// ============================================================================

type FounderLeadAdminHandler struct {
	service *service.FounderLeadService
}

func NewFounderLeadAdminHandler(service *service.FounderLeadService) *FounderLeadAdminHandler {
	return &FounderLeadAdminHandler{service: service}
}

// ============================================================================
// LIST - GET /api/v1/admin/founders-leads
// ============================================================================

func (h *FounderLeadAdminHandler) List(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "25"))
	status := c.Query("status", "")
	search := c.Query("search", "")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 25
	}

	result, err := h.service.ListLeads(c.Context(), postgres.ListFounderLeadsFilter{
		Status: status,
		Search: search,
		Page:   page,
		Limit:  limit,
	})
	if err != nil {
		logger.Error("Failed to list founder leads", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to fetch leads",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    result.Leads,
		"pagination": fiber.Map{
			"total":    result.Total,
			"page":     result.Page,
			"pages":    result.Pages,
			"limit":    result.Limit,
			"has_next": result.HasNext,
			"has_prev": result.HasPrev,
		},
	})
}

// ============================================================================
// STATS - GET /api/v1/admin/founders-leads/stats
// ============================================================================

func (h *FounderLeadAdminHandler) GetStats(c *fiber.Ctx) error {
	stats, err := h.service.GetStats(c.Context())
	if err != nil {
		logger.Error("Failed to get founder leads stats", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to fetch stats",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    stats,
	})
}

// ============================================================================
// GET ONE - GET /api/v1/admin/founders-leads/:id
// ============================================================================

func (h *FounderLeadAdminHandler) GetOne(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid ID",
		})
	}

	lead, err := h.service.GetLead(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "Lead not found",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    lead,
	})
}

// ============================================================================
// UPDATE STATUS - PATCH /api/v1/admin/founders-leads/:id/status
// ============================================================================

func (h *FounderLeadAdminHandler) UpdateStatus(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid ID",
		})
	}

	var req domain.PatchLeadStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	lead, err := h.service.UpdateLeadStatus(c.Context(), id, req.Status)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to update status",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    lead,
	})
}

// ============================================================================
// SEND EMAIL - POST /api/v1/admin/founders-leads/:id/email
// ============================================================================

func (h *FounderLeadAdminHandler) SendEmail(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid ID",
		})
	}

	var req domain.SendLeadEmailRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	if err := h.service.SendFollowUp(c.Context(), id, req.Subject, req.Message); err != nil {
		logger.Error("Failed to send email", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to send email",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Email sent successfully",
	})
}

// ============================================================================
// DELETE - DELETE /api/v1/admin/founders-leads/:id
// ============================================================================

func (h *FounderLeadAdminHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid ID",
		})
	}

	if err := h.service.DeleteLead(c.Context(), id); err != nil {
		logger.Error("Failed to delete lead", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to delete lead",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Lead deleted successfully",
	})
}
