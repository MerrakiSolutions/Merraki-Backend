package main

import (
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
	"go.uber.org/zap"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// Initialize logger
	if err := logger.InitLogger(cfg.Logging.Level, cfg.Logging.Format, cfg.Logging.Output); err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}
	defer logger.Sync()

	logger.Info("Starting Merraki API Server",
		zap.String("env", cfg.Server.Environment),
		zap.Int("port", cfg.Server.Port),
		zap.String("version", cfg.Server.APIVersion),
	)

	// Initialize database
	db, err := postgres.NewDatabase(cfg)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Initialize Redis
	redisClient, err := redis.NewRedisClient(cfg)
	if err != nil {
		logger.Fatal("Failed to connect to Redis", zap.Error(err))
	}

	// ============================================
	// INITIALIZE REPOSITORIES
	// ============================================
	adminRepo := postgres.NewAdminRepository(db)
	sessionRepo := postgres.NewSessionRepository(db)
	templateRepo := postgres.NewTemplateRepository(db)
	categoryRepo := postgres.NewCategoryRepository(db)
	orderRepo := postgres.NewOrderRepository(db)
	
	// Blog repositories
	blogPostRepo := postgres.NewBlogPostRepository(db)
	blogAuthorRepo := postgres.NewBlogAuthorRepository(db)
	blogCategoryRepo := postgres.NewBlogCategoryRepository(db)
	
	newsletterRepo := postgres.NewNewsletterRepository(db)
	contactRepo := postgres.NewContactRepository(db)
	testRepo := postgres.NewTestRepository(db)
	calculatorRepo := postgres.NewCalculatorRepository(db)
	activityLogRepo := postgres.NewActivityLogRepository(db)

	// ============================================
	// INITIALIZE SERVICES
	// ============================================
	storageService, err := service.NewStorageService(cfg)
	if err != nil {
		logger.Fatal("Failed to initialize storage service", zap.Error(err))
	}

	_ = service.NewPDFService(storageService)
	_ = service.NewCurrencyService(cfg)

	emailService := service.NewEmailService(cfg)
	paymentService := service.NewPaymentService(cfg)

	authService, err := service.NewAuthService(adminRepo, sessionRepo, activityLogRepo, cfg)
	if err != nil {
		logger.Fatal("Failed to initialize auth service", zap.Error(err))
	}

	adminService := service.NewAdminService(adminRepo, activityLogRepo)
	categoryService := service.NewCategoryService(categoryRepo, activityLogRepo)
	templateService := service.NewTemplateService(templateRepo, categoryRepo, activityLogRepo)
	orderService := service.NewOrderService(orderRepo, templateRepo, activityLogRepo, paymentService, emailService)
	
	// Blog services
	blogAuthorService := service.NewBlogAuthorService(blogAuthorRepo, activityLogRepo)
	blogCategoryService := service.NewBlogCategoryService(blogCategoryRepo, activityLogRepo)
	blogPostService := service.NewBlogPostService(blogPostRepo, blogAuthorRepo, blogCategoryRepo, activityLogRepo)
	
	newsletterService := service.NewNewsletterService(newsletterRepo, emailService)
	contactService := service.NewContactService(contactRepo, activityLogRepo, emailService)
	testService := service.NewTestService(testRepo, activityLogRepo, emailService)
	calculatorService := service.NewCalculatorService(calculatorRepo)
	dashboardService := service.NewDashboardService(db.Pool)

	// ============================================
	// INITIALIZE HANDLERS
	// ============================================
	// Admin Handlers
	adminHandlersStruct := &routes.AdminHandlers{
		Auth:         adminHandlers.NewAuthHandler(authService),
		Dashboard:    adminHandlers.NewDashboardHandler(dashboardService),
		Template:     adminHandlers.NewTemplateHandler(templateService),
		Order:        adminHandlers.NewOrderHandler(orderService),
		BlogPost:     adminHandlers.NewBlogPostHandler(blogPostService),
		BlogAuthor:   adminHandlers.NewBlogAuthorHandler(blogAuthorService),
		BlogCategory: adminHandlers.NewBlogCategoryHandler(blogCategoryService),
		Contact:      adminHandlers.NewContactHandler(contactService),
		Test:         adminHandlers.NewTestHandler(testService),
		Calculator:   adminHandlers.NewCalculatorHandler(calculatorService),
		AdminUser:    adminHandlers.NewAdminUserHandler(adminService),
	}

	// Public Handlers
	publicHandlersStruct := &routes.PublicHandlers{
		Template:   publicHandlers.NewTemplateHandler(templateService, categoryService),
		Order:      publicHandlers.NewOrderHandler(orderService),
		Calculator: publicHandlers.NewCalculatorHandler(calculatorService),
		Blog:       publicHandlers.NewBlogHandler(blogPostService, blogAuthorService, blogCategoryService),
		Newsletter: publicHandlers.NewNewsletterHandler(newsletterService),
		Contact:    publicHandlers.NewContactHandler(contactService),
		Test:       publicHandlers.NewTestHandler(testService),
		Utility:    publicHandlers.NewUtilityHandler(db, redisClient),
	}

	// ============================================
	// INITIALIZE FIBER APP
	// ============================================
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			logger.Error("Fiber error",
				zap.Error(err),
				zap.String("path", c.Path()),
				zap.String("method", c.Method()),
			)

			if cfg.Server.Environment == "development" {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"success": false,
					"message": err.Error(),
					"code":    "INTERNAL_ERROR",
					"path":    c.Path(),
				})
			}

			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": "Internal server error",
				"code":    "INTERNAL_ERROR",
			})
		},
		BodyLimit:             10 * 1024 * 1024,
		ReadTimeout:           30 * time.Second,
		WriteTimeout:          30 * time.Second,
		IdleTimeout:           120 * time.Second,
		DisableStartupMessage: false,
		AppName:               "Merraki API v1.0.0",
	})

	// ============================================
	// GLOBAL MIDDLEWARE
	// ============================================
	app.Use(middleware.Recovery())
	app.Use(middleware.RequestID())
	app.Use(middleware.Logger())
	app.Use(middleware.Security())
	app.Use(middleware.CORS(cfg))
	app.Use(middleware.RateLimit(100, 1*time.Minute))

	// ============================================
	// ROOT & HEALTH ENDPOINTS
	// ============================================
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"success": true,
			"message": "Merraki API Server",
			"version": "1.0.0",
			"docs":    "/api/v1/docs",
		})
	})

	app.Get("/health", publicHandlersStruct.Utility.Health)

	// ============================================
	// API ROUTES
	// ============================================
	api := app.Group("/api/v1")

	// Setup Public Routes
	routes.SetupPublicRoutes(api, publicHandlersStruct)

	// Setup Admin Routes
	routes.SetupAdminRoutes(api, adminHandlersStruct, cfg)

	// ============================================
	// 404 HANDLER
	// ============================================
	app.Use(func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Route not found",
			"code":    "NOT_FOUND",
			"path":    c.Path(),
		})
	})

	// ============================================
	// START SERVER
	// ============================================
	go func() {
		addr := fmt.Sprintf(":%d", cfg.Server.Port)
		logger.Info("🚀 Server starting",
			zap.String("address", addr),
			zap.String("env", cfg.Server.Environment),
		)
		logger.Info("📚 API Documentation: http://localhost:"+fmt.Sprint(cfg.Server.Port)+"/api/v1/docs")
		logger.Info("❤️  Health Check: http://localhost:"+fmt.Sprint(cfg.Server.Port)+"/health")

		if err := app.Listen(addr); err != nil {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// ============================================
	// GRACEFUL SHUTDOWN
	// ============================================
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	logger.Info("⚠️  Shutting down server...")
	if err := app.Shutdown(); err != nil {
		logger.Error("Server shutdown error", zap.Error(err))
	}

	logger.Info("✅ Server stopped gracefully")
}