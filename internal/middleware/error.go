package middleware

import (
	"github.com/gofiber/fiber/v2"
	apperrors "github.com/merraki/merraki-backend/internal/pkg/errors"
	"github.com/merraki/merraki-backend/internal/pkg/logger"
	"github.com/merraki/merraki-backend/internal/pkg/response"
	"go.uber.org/zap"
)

func ErrorHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		err := c.Next()

		if err != nil {
			// Log error
			logger.Error("Request error",
				zap.String("path", c.Path()),
				zap.String("method", c.Method()),
				zap.Error(err),
			)

			// Handle custom app errors
			if appErr, ok := err.(*apperrors.AppError); ok {
				return response.Error(c, appErr)
			}

			// Handle fiber errors
			if fiberErr, ok := err.(*fiber.Error); ok {
				return c.Status(fiberErr.Code).JSON(fiber.Map{
					"success": false,
					"message": fiberErr.Message,
					"code":    "FIBER_ERROR",
				})
			}

			// Generic error
			return response.Error(c, apperrors.ErrInternalServer)
		}

		return nil
	}
}