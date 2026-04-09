package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/merraki/merraki-backend/internal/config"
	adminHandlers "github.com/merraki/merraki-backend/internal/handler/admin"
	publicHandlers "github.com/merraki/merraki-backend/internal/handler/public"
	"github.com/merraki/merraki-backend/internal/middleware"
	"github.com/merraki/merraki-backend/internal/pkg/logger"
	"github.com/merraki/merraki-backend/internal/repository/postgres"
	"github.com/merraki/merraki-backend/internal/repository/redis"
	"github.com/merraki/merraki-backend/internal/routes"
	"github.com/merraki/merraki-backend/internal/service"
	"github.com/merraki/merraki-backend/internal/worker"
	"go.uber.org/zap"
)

func main() {
	// ========================================================================
	// LOAD CONFIGURATION
	// ========================================================================
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// ========================================================================
	// INITIALIZE LOGGER
	// ========================================================================
	if err := logger.InitLogger(cfg.Logging.Level, cfg.Logging.Format, cfg.Logging.Output); err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}
	defer logger.Sync()

	logger.Info("🚀 Starting Merraki API Server",
		zap.String("env", cfg.Server.Environment),
		zap.Int("port", cfg.Server.Port),
		zap.String("version", cfg.Server.APIVersion),
	)

	// ========================================================================
	// INITIALIZE DATABASE
	// ========================================================================
	db, err := postgres.NewDatabase(cfg)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	logger.Info("✅ Database connected successfully")

	// ========================================================================
	// INITIALIZE REDIS
	// ========================================================================
	redisClient, err := redis.NewRedisClient(cfg)
	if err != nil {
		logger.Fatal("Failed to connect to Redis", zap.Error(err))
	}

	logger.Info("✅ Redis connected successfully")

	// ========================================================================
	// INITIALIZE REPOSITORIES
	// ========================================================================

	// Auth & Admin
	adminRepo := postgres.NewAdminRepository(db)
	sessionRepo := postgres.NewSessionRepository(db)
	activityLogRepo := postgres.NewActivityLogRepository(db)

	// Marketplace (NEW SYSTEM)
	categoryRepo := postgres.NewCategoryRepository(db.DB)
	templateRepo := postgres.NewTemplateRepository(db.DB)
	orderRepo := postgres.NewOrderRepository(db.DB)
	orderItemRepo := postgres.NewOrderItemRepository(db.DB)
	paymentRepo := postgres.NewPaymentRepository(db.DB)
	webhookRepo := postgres.NewPaymentWebhookRepository(db.DB)
	downloadTokenRepo := postgres.NewDownloadTokenRepository(db.DB)
	downloadRepo := postgres.NewDownloadRepository(db.DB)
	idempotencyRepo := postgres.NewIdempotencyKeyRepository(db.DB)
	transitionRepo := postgres.NewOrderStateTransitionRepository(db.DB)
	circuitBreakerRepo := postgres.NewCircuitBreakerRepository(db.DB)
	jobRepo := postgres.NewBackgroundJobRepository(db.DB)

	// Blog System (EXISTING)
	blogPostRepo := postgres.NewBlogPostRepository(db)
	blogAuthorRepo := postgres.NewBlogAuthorRepository(db)
	blogCategoryRepo := postgres.NewBlogCategoryRepository(db)

	// Newsletter & Contact (EXISTING)
	newsletterRepo := postgres.NewNewsletterRepository(db)
	contactRepo := postgres.NewContactRepository(db)

	logger.Info("✅ Repositories initialized")

	// ========================================================================
	// INITIALIZE SERVICES
	// ========================================================================

	// Storage & Files
	storageService, err := service.NewStorageService(cfg)
	if err != nil {
		logger.Fatal("Failed to initialize storage service", zap.Error(err))
	}

	// PDF Service
	pdfService := service.NewPDFService(storageService)
	// Email Service
	emailService := service.NewEmailService(cfg)

	// Payment Service (NEW - with circuit breaker)
	paymentService := service.NewPaymentService(cfg, webhookRepo, circuitBreakerRepo)

	// Auth Service
	authService, err := service.NewAuthService(adminRepo, sessionRepo, activityLogRepo, cfg)
	if err != nil {
		logger.Fatal("Failed to initialize auth service", zap.Error(err))
	}

	// Admin Service
	adminService := service.NewAdminService(adminRepo, activityLogRepo)

	// Marketplace Services (NEW)
	categoryService := service.NewCategoryService(categoryRepo, activityLogRepo)
	templateService := service.NewTemplateService(templateRepo, categoryRepo, activityLogRepo)

	orderService := service.NewOrderService(
		orderRepo,
		orderItemRepo,
		templateRepo,
		paymentRepo,
		idempotencyRepo,
		transitionRepo,
		activityLogRepo,
		paymentService,
		emailService,
		jobRepo,
	)

	downloadTokenService := service.NewDownloadTokenService(
		downloadTokenRepo,
		downloadRepo,
		orderRepo,
		orderItemRepo,
		templateRepo,
		storageService,
	)

	// Blog Services (EXISTING)
	blogAuthorService := service.NewBlogAuthorService(blogAuthorRepo, activityLogRepo)
	blogCategoryService := service.NewBlogCategoryService(blogCategoryRepo, activityLogRepo)
	blogPostService := service.NewBlogPostService(blogPostRepo, blogAuthorRepo, blogCategoryRepo, activityLogRepo)

	// Newsletter & Contact Services (EXISTING)
	newsletterService := service.NewNewsletterService(newsletterRepo, emailService)
	contactService := service.NewContactService(contactRepo, activityLogRepo, emailService)

	// Dashboard Service (EXISTING)
	dashboardService := service.NewDashboardService(db.Pool)

	logger.Info("✅ Services initialized")

	// ========================================================================
	// INITIALIZE BACKGROUND WORKERS (NEW - Optional embedded mode)
	// ========================================================================

	jobProcessor := worker.NewJobProcessor(
		jobRepo,
		orderRepo,
		orderItemRepo,
		paymentRepo,
		webhookRepo,
		downloadTokenRepo,
		idempotencyRepo,
		emailService,
		downloadTokenService,
		paymentService,
		pdfService,
		storageService,
		"worker-api-1",
	)

	scheduledRunner := worker.NewScheduledJobRunner(jobRepo)

	// Start workers in background (optional - can run cmd/worker/main.go separately)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := jobProcessor.Start(ctx); err != nil {
			logger.Error("Job processor error", zap.Error(err))
		}
	}()

	go func() {
		if err := scheduledRunner.Start(ctx); err != nil {
			logger.Error("Scheduled runner error", zap.Error(err))
		}
	}()

	logger.Info("✅ Background workers started")

	// ========================================================================
	// INITIALIZE HANDLERS
	// ========================================================================

	// Public Handlers
	publicHandlersStruct := &routes.PublicHandlers{
		Template:   publicHandlers.NewTemplateHandler(templateService, categoryService),
		Order:      publicHandlers.NewOrderHandler(orderService),
		Checkout:   publicHandlers.NewCheckoutHandler(orderService, paymentService),
		Download:   publicHandlers.NewDownloadHandler(downloadTokenService),
		Blog:       publicHandlers.NewBlogHandler(blogPostService, blogAuthorService, blogCategoryService),
		Newsletter: publicHandlers.NewNewsletterHandler(newsletterService),
		Contact:    publicHandlers.NewContactHandler(contactService),
		Utility:    publicHandlers.NewUtilityHandler(db, redisClient),
	}

	// Admin Handlers
	adminHandlersStruct := &routes.AdminHandlers{
		Auth:         adminHandlers.NewAuthHandler(authService),
		Dashboard:    adminHandlers.NewDashboardHandler(dashboardService),
		Order:        adminHandlers.NewOrderHandler(orderService),
		Template:     adminHandlers.NewTemplateHandler(templateService, storageService),
		Category:     adminHandlers.NewCategoryHandler(categoryService),
		BlogPost:     adminHandlers.NewBlogPostHandler(blogPostService),
		BlogAuthor:   adminHandlers.NewBlogAuthorHandler(blogAuthorService),
		BlogCategory: adminHandlers.NewBlogCategoryHandler(blogCategoryService),
		Newsletter:   adminHandlers.NewNewsletterHandler(newsletterService),
		Contact:      adminHandlers.NewContactHandler(contactService),
		AdminUser:    adminHandlers.NewAdminUserHandler(adminService),
	}

	logger.Info("✅ Handlers initialized")

	// ========================================================================
	// INITIALIZE FIBER APP
	// ========================================================================

	app := fiber.New(fiber.Config{
		AppName:               "Merraki API v1.0.0",
		ServerHeader:          "Merraki",
		ErrorHandler:          customErrorHandler(cfg),
		BodyLimit:             10 * 1024 * 1024, // 10MB
		ReadTimeout:           30 * time.Second,
		WriteTimeout:          30 * time.Second,
		IdleTimeout:           120 * time.Second,
		DisableStartupMessage: false,
	})

	// ========================================================================
	// GLOBAL MIDDLEWARE
	// ========================================================================
	app.Use(middleware.Recovery())
	app.Use(middleware.RequestID())
	app.Use(middleware.Logger())
	app.Use(middleware.Security())
	app.Use(middleware.CORS(cfg))
	app.Use(middleware.RateLimit(100, 1*time.Minute))

	// ========================================================================
	// ROOT & HEALTH ENDPOINTS
	// ========================================================================
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"success": true,
			"message": "Merraki API Server",
			"version": "1.0.0",
			"docs":    "/api/v1/docs",
		})
	})

	app.Get("/health", publicHandlersStruct.Utility.Health)

	// ========================================================================
	// API ROUTES
	// ========================================================================
	api := app.Group("/api/v1")

	// Setup Public Routes
	routes.SetupPublicRoutes(api, publicHandlersStruct)

	// Setup Admin Routes
	routes.SetupAdminRoutes(api, adminHandlersStruct, cfg)

	// ========================================================================
	// 404 HANDLER
	// ========================================================================
	app.Use(func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Route not found",
			"code":    "NOT_FOUND",
			"path":    c.Path(),
		})
	})

	logger.Info("✅ Routes configured")

	// ========================================================================
	// START SERVER
	// ========================================================================
	go func() {
		addr := fmt.Sprintf(":%d", cfg.Server.Port)
		logger.Info("🌐 Server running",
			zap.String("address", fmt.Sprintf("http://localhost:%d", cfg.Server.Port)),
			zap.String("env", cfg.Server.Environment),
		)
		logger.Info("📚 API Documentation: http://localhost:" + fmt.Sprint(cfg.Server.Port) + "/api/v1/docs")
		logger.Info("❤️  Health Check: http://localhost:" + fmt.Sprint(cfg.Server.Port) + "/health")

		if err := app.Listen(addr); err != nil {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// ========================================================================
	// GRACEFUL SHUTDOWN
	// ========================================================================

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	logger.Info("⚠️  Shutting down server...")

	// Stop workers
	cancel()
	jobProcessor.Stop()
	scheduledRunner.Stop()

	// Shutdown server
	if err := app.ShutdownWithTimeout(30 * time.Second); err != nil {
		logger.Error("Server shutdown error", zap.Error(err))
	}

	logger.Info("✅ Server stopped gracefully")
}

// ============================================================================
// CUSTOM ERROR HANDLER
// ============================================================================

func customErrorHandler(cfg *config.Config) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		logger.Error("Request error",
			zap.Error(err),
			zap.String("path", c.Path()),
			zap.String("method", c.Method()),
		)

		code := fiber.StatusInternalServerError
		message := "Internal server error"

		if e, ok := err.(*fiber.Error); ok {
			code = e.Code
			message = e.Message
		}

		response := fiber.Map{
			"success": false,
			"error":   message,
		}

		if cfg.Server.Environment == "development" {
			response["details"] = err.Error()
			response["path"] = c.Path()
		}

		return c.Status(code).JSON(response)
	}
}
