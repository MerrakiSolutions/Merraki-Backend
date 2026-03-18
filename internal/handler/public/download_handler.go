package public

import (
	"github.com/gofiber/fiber/v2"
	"github.com/merraki/merraki-backend/internal/pkg/logger"
	"github.com/merraki/merraki-backend/internal/service"
	"go.uber.org/zap"
)

// ============================================================================
// DOWNLOAD HANDLER - Secure file downloads
// ============================================================================

type DownloadHandler struct {
	downloadTokenService *service.DownloadTokenService
}

func NewDownloadHandler(downloadTokenService *service.DownloadTokenService) *DownloadHandler {
	return &DownloadHandler{
		downloadTokenService: downloadTokenService,
	}
}

// ============================================================================
// INITIATE DOWNLOAD
// ============================================================================

// GET /api/v1/download?token=xxx&email=xxx
func (h *DownloadHandler) InitiateDownload(c *fiber.Ctx) error {
	token := c.Query("token")
	email := c.Query("email")

	if token == "" || email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Token and email are required",
		})
	}

	// Build service request
	serviceReq := &service.InitiateDownloadRequest{
		Token:     token,
		Email:     email,
		IPAddress: c.IP(),
		UserAgent: string(c.Request().Header.UserAgent()),
		Country:   c.Get("CF-IPCountry"), // Cloudflare header
	}

	// Get download URL
	response, err := h.downloadTokenService.InitiateDownload(c.Context(), serviceReq)
	if err != nil {
		logger.Error("Download initiation failed",
			zap.String("token", token[:16]+"..."),
			zap.String("email", email),
			zap.Error(err),
		)
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Invalid or expired download link",
		})
	}

	// Redirect to signed URL
	return c.Redirect(response.DownloadURL, fiber.StatusTemporaryRedirect)
}

// ============================================================================
// GET DOWNLOAD INFO (without redirect)
// ============================================================================

// POST /api/v1/download/info
func (h *DownloadHandler) GetDownloadInfo(c *fiber.Ctx) error {
	var req struct {
		Token string `json:"token" validate:"required"`
		Email string `json:"email" validate:"required,email"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Build service request
	serviceReq := &service.InitiateDownloadRequest{
		Token:     req.Token,
		Email:     req.Email,
		IPAddress: c.IP(),
		UserAgent: string(c.Request().Header.UserAgent()),
		Country:   c.Get("CF-IPCountry"),
	}

	// Get download URL
	response, err := h.downloadTokenService.InitiateDownload(c.Context(), serviceReq)
	if err != nil {
		logger.Error("Download initiation failed", zap.Error(err))
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Invalid or expired download link",
		})
	}

	return c.JSON(fiber.Map{
		"download_url": response.DownloadURL,
		"expires_at":   response.ExpiresAt,
		"file_name":    response.FileName,
		"file_size":    response.FileSize,
	})
}

// ============================================================================
// GET DOWNLOADS BY EMAIL
// ============================================================================

// GET /api/v1/downloads/by-email?email=xxx
func (h *DownloadHandler) GetDownloadsByEmail(c *fiber.Ctx) error {
	email := c.Query("email")
	if email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Email is required",
		})
	}

	tokens, err := h.downloadTokenService.GetTokensByEmail(c.Context(), email)
	if err != nil {
		logger.Error("Failed to get downloads", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get downloads",
		})
	}

	return c.JSON(fiber.Map{
		"downloads": tokens,
	})
}