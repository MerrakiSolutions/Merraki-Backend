package admin

import (
	"github.com/gofiber/fiber/v2"
	"github.com/merraki/merraki-backend/internal/pkg/logger"
	"github.com/merraki/merraki-backend/internal/pkg/response"
	"github.com/merraki/merraki-backend/internal/service"
	"go.uber.org/zap"
)

type DashboardHandler struct {
	dashboardService *service.DashboardService
}

func NewDashboardHandler(dashboardService *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{dashboardService: dashboardService}
}

// GetStats — primary endpoint consumed by the frontend dashboard page.
// GET /api/v1/admin/dashboard/stats
func (h *DashboardHandler) GetStats(c *fiber.Ctx) error {
	stats, err := h.dashboardService.GetStats(c.Context())
	if err != nil {
		logger.Error("Failed to get dashboard stats", zap.Error(err))
		return response.Error(c, err)
	}
	return response.SuccessData(c, stats)
}

// GetSummary — backward-compat alias for GetStats.
// GET /api/v1/admin/dashboard/summary
func (h *DashboardHandler) GetSummary(c *fiber.Ctx) error {
	stats, err := h.dashboardService.GetStats(c.Context())
	if err != nil {
		logger.Error("Failed to get dashboard summary", zap.Error(err))
		return response.Error(c, err)
	}
	return response.SuccessData(c, stats)
}

// GetActivity — live activity feed (last 50 admin actions).
// GET /api/v1/admin/dashboard/activity
func (h *DashboardHandler) GetActivity(c *fiber.Ctx) error {
	logs, err := h.dashboardService.GetActivity(c.Context())
	if err != nil {
		logger.Error("Failed to get activity logs", zap.Error(err))
		return response.Error(c, err)
	}
	return response.SuccessData(c, fiber.Map{
		"activities": logs,
	})
}

// GetCharts — returns only chart slices for dedicated chart requests.
// GET /api/v1/admin/dashboard/charts
func (h *DashboardHandler) GetCharts(c *fiber.Ctx) error {
	stats, err := h.dashboardService.GetStats(c.Context())
	if err != nil {
		logger.Error("Failed to get chart data", zap.Error(err))
		return response.Error(c, err)
	}
	return response.SuccessData(c, fiber.Map{
		"monthlyRevenue": stats["monthlyRevenue"],
		"dailyRevenue":   stats["dailyRevenue"],
		"ordersByStatus": stats["ordersByStatus"],
	})
}

// GetNotifications — placeholder until notification system is built.
// GET /api/v1/admin/dashboard/notifications
func (h *DashboardHandler) GetNotifications(c *fiber.Ctx) error {
	return response.SuccessData(c, fiber.Map{
		"notifications": []interface{}{},
		"unread_count":  0,
	})
}

// MarkNotificationRead — placeholder.
// PATCH /api/v1/admin/dashboard/notifications/:id/read
func (h *DashboardHandler) MarkNotificationRead(c *fiber.Ctx) error {
	return response.Success(c, "Notification marked as read", nil)
}

// GlobalSearch — placeholder for cross-entity search.
// GET /api/v1/admin/dashboard/search?q=...
func (h *DashboardHandler) GlobalSearch(c *fiber.Ctx) error {
	query := c.Query("q")
	return response.SuccessData(c, fiber.Map{
		"query":   query,
		"results": []interface{}{},
	})
}

// GetSettings — placeholder for admin settings.
// GET /api/v1/admin/dashboard/settings
func (h *DashboardHandler) GetSettings(c *fiber.Ctx) error {
	return response.SuccessData(c, fiber.Map{
		"settings": fiber.Map{},
	})
}

// UpdateSettings — placeholder.
// PUT /api/v1/admin/dashboard/settings
func (h *DashboardHandler) UpdateSettings(c *fiber.Ctx) error {
	return response.Success(c, "Settings updated successfully", nil)
}