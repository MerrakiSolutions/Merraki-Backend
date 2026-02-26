package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/merraki/merraki-backend/internal/pkg/logger"
	"go.uber.org/zap"
)

func Recovery() fiber.Handler {
	return recover.New(recover.Config{
		EnableStackTrace: true,
		StackTraceHandler: func(c *fiber.Ctx, e interface{}) {
			logger.Error("Panic recovered",
				zap.String("path", c.Path()),
				zap.String("method", c.Method()),
				zap.Any("error", e),
			)
		},
	})
}