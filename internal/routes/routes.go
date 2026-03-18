package routes

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/merraki/merraki-backend/internal/config"
)

// ============================================================================
// SETUP ALL ROUTES - Main entry point
// ============================================================================

func SetupRoutes(
	app *fiber.App,
	cfg *config.Config,
	publicHandlers *PublicHandlers,
	adminHandlers *AdminHandlers,
) {
	// ========================================================================
	// GLOBAL MIDDLEWARE
	// ========================================================================

	// Recover from panics
	app.Use(recover.New(recover.Config{
		EnableStackTrace: cfg.Server.Environment == "development",
	}))

	// Request logger
	app.Use(logger.New(logger.Config{
		Format:     "${time} | ${status} | ${latency} | ${method} ${path}\n",
		TimeFormat: "2006-01-02 15:04:05",
	}))

	// CORS
	app.Use(cors.New(cors.Config{
		AllowOrigins:     strings.Join(cfg.CORS.AllowedOrigins, ","),
		AllowMethods:     strings.Join(cfg.CORS.AllowedMethods, ","),
		AllowHeaders:     strings.Join(cfg.CORS.AllowedHeaders, ","),
		AllowCredentials: true,
		MaxAge:           86400,
	}))

	// ========================================================================
	// ROOT ROUTE
	// ========================================================================

	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"success": true,
			"message": "Merraki API Server",
			"version": "1.0.0",
			"status":  "running",
			"docs":    "/api/v1/docs",
		})
	})

	// ========================================================================
	// HEALTH CHECK
	// ========================================================================

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"service": "merraki-api",
			"version": "1.0.0",
		})
	})

	// ========================================================================
	// API VERSION GROUP
	// ========================================================================

	api := app.Group("/api/v1")

	// Setup public routes
	SetupPublicRoutes(api, publicHandlers)

	// Setup admin routes
	SetupAdminRoutes(api, adminHandlers, cfg)

	// ========================================================================
	// 404 HANDLER
	// ========================================================================

	app.Use(func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "Route not found",
			"path":    c.Path(),
		})
	})
}
