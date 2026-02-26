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

	// Initialize repositories
	adminRepo := postgres.NewAdminRepository(db)
	sessionRepo := postgres.NewSessionRepository(db)
	templateRepo := postgres.NewTemplateRepository(db)
	categoryRepo := postgres.NewCategoryRepository(db)
	orderRepo := postgres.NewOrderRepository(db)
	blogRepo := postgres.NewBlogRepository(db)
	newsletterRepo := postgres.NewNewsletterRepository(db)
	contactRepo := postgres.NewContactRepository(db)
	testRepo := postgres.NewTestRepository(db)
	calculatorRepo := postgres.NewCalculatorRepository(db)
	activityLogRepo := postgres.NewActivityLogRepository(db)

	// Initialize services
	storageService, err := service.NewStorageService(cfg)
	if err != nil {
		logger.Fatal("Failed to initialize storage service", zap.Error(err))
	}

	// PDF and Currency services - initialized but used in future features
	_ = service.NewPDFService(storageService) // Will be used for PDF generation
	_ = service.NewCurrencyService(cfg)       // Will be used for currency conversion

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
	blogService := service.NewBlogService(blogRepo, activityLogRepo)
	newsletterService := service.NewNewsletterService(newsletterRepo, emailService)
	contactService := service.NewContactService(contactRepo, activityLogRepo, emailService)
	testService := service.NewTestService(testRepo, activityLogRepo, emailService)
	calculatorService := service.NewCalculatorService(calculatorRepo)
	dashboardService := service.NewDashboardService(db.Pool)

	// Initialize handlers - Admin
	adminAuthHandler := adminHandlers.NewAuthHandler(authService)
	adminDashboardHandler := adminHandlers.NewDashboardHandler(dashboardService)
	adminTemplateHandler := adminHandlers.NewTemplateHandler(templateService)
	adminCategoryHandler := adminHandlers.NewCategoryHandler(categoryService)
	adminOrderHandler := adminHandlers.NewOrderHandler(orderService)
	adminBlogHandler := adminHandlers.NewBlogHandler(blogService)
	adminNewsletterHandler := adminHandlers.NewNewsletterHandler(newsletterService)
	adminContactHandler := adminHandlers.NewContactHandler(contactService)
	adminTestHandler := adminHandlers.NewTestHandler(testService)
	adminCalculatorHandler := adminHandlers.NewCalculatorHandler(calculatorService)
	adminUserHandler := adminHandlers.NewAdminUserHandler(adminService)

	// Initialize handlers - Public
	publicTemplateHandler := publicHandlers.NewTemplateHandler(templateService, categoryService)
	publicOrderHandler := publicHandlers.NewOrderHandler(orderService)
	publicCalculatorHandler := publicHandlers.NewCalculatorHandler(calculatorService)
	publicBlogHandler := publicHandlers.NewBlogHandler(blogService, categoryService)
	publicNewsletterHandler := publicHandlers.NewNewsletterHandler(newsletterService)
	publicContactHandler := publicHandlers.NewContactHandler(contactService)
	publicTestHandler := publicHandlers.NewTestHandler(testService)
	publicUtilityHandler := publicHandlers.NewUtilityHandler(db, redisClient)

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			logger.Error("Fiber error",
				zap.Error(err),
				zap.String("path", c.Path()),
				zap.String("method", c.Method()),
			)

			// Show detailed error in development
			if cfg.Server.Environment == "development" {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"success": false,
					"message": err.Error(), // SHOW ACTUAL ERROR
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
		BodyLimit:             10 * 1024 * 1024, // 10MB
		ReadTimeout:           30 * time.Second,
		WriteTimeout:          30 * time.Second,
		IdleTimeout:           120 * time.Second,
		DisableStartupMessage: false,
		AppName:               "Merraki API v1.0.0",
	})

	// Global middleware
	app.Use(middleware.Recovery())
	app.Use(middleware.RequestID())
	app.Use(middleware.Logger())
	app.Use(middleware.Security())
	app.Use(middleware.CORS(cfg))
	app.Use(middleware.RateLimit(100, 1*time.Minute))

	// Root endpoint
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"success": true,
			"message": "Merraki API Server",
			"version": "1.0.0",
			"docs":    "/api/v1/docs",
		})
	})

	// Health check
	app.Get("/health", publicUtilityHandler.Health)

	// API v1
	api := app.Group("/api/v1")

	// ============================================
	// PUBLIC ROUTES
	// ============================================
	public := api.Group("/public")
	{
		// Templates
		templates := public.Group("/templates")
		templates.Get("/", publicTemplateHandler.GetAll)
		templates.Get("/featured", publicTemplateHandler.GetFeatured)
		templates.Get("/popular", publicTemplateHandler.GetPopular)
		templates.Get("/search", publicTemplateHandler.Search)
		templates.Get("/:slug", publicTemplateHandler.GetBySlug)

		// Categories
		public.Get("/categories", publicTemplateHandler.GetCategories)

		// Orders
		orders := public.Group("/orders")
		orders.Post("/", publicOrderHandler.Create)
		orders.Post("/verify", publicOrderHandler.VerifyPayment)
		orders.Get("/lookup", publicOrderHandler.Lookup)
		orders.Get("/download/:orderNumber", publicOrderHandler.Download)
		orders.Post("/webhook", publicOrderHandler.Webhook) // Razorpay webhook

		// Calculators
		calculators := public.Group("/calculators")
		calculators.Post("/valuation", publicCalculatorHandler.CalculateValuation)
		calculators.Post("/breakeven", publicCalculatorHandler.CalculateBreakeven)
		calculators.Post("/save", publicCalculatorHandler.SaveResult)
		calculators.Get("/results", publicCalculatorHandler.GetResults)
		calculators.Post("/export-pdf", publicCalculatorHandler.ExportPDF)

		// Blog
		blog := public.Group("/blog")
		blog.Get("/posts", publicBlogHandler.GetAll)
		blog.Get("/posts/featured", publicBlogHandler.GetFeatured)
		blog.Get("/posts/popular", publicBlogHandler.GetPopular)
		blog.Get("/posts/search", publicBlogHandler.Search)
		blog.Get("/posts/:slug", publicBlogHandler.GetBySlug)
		blog.Get("/categories", publicBlogHandler.GetCategories)

		// Newsletter
		newsletter := public.Group("/newsletter")
		newsletter.Post("/subscribe", publicNewsletterHandler.Subscribe)
		newsletter.Post("/unsubscribe", publicNewsletterHandler.Unsubscribe)
		newsletter.Get("/unsubscribe", publicNewsletterHandler.UnsubscribeGET)

		// Contact
		public.Post("/contact", publicContactHandler.Create)

		// Test
		test := public.Group("/test")
		test.Get("/questions", publicTestHandler.GetQuestions)
		test.Post("/submit", publicTestHandler.Submit)
		test.Get("/results/:testNumber", publicTestHandler.GetResults)

		// Utility
		public.Get("/currency/convert", publicUtilityHandler.CurrencyConvert)
	}

	// ============================================
	// ADMIN ROUTES
	// ============================================
	admin := api.Group("/admin")
	{
		// Auth routes (no authentication required)
		auth := admin.Group("/auth")
		auth.Post("/login", adminAuthHandler.Login)
		auth.Post("/refresh", adminAuthHandler.RefreshToken)

		// Protected admin routes
		protected := admin.Use(middleware.AdminAuth(cfg))

		// ========== Auth Management ==========
		authRoutes := protected.Group("/auth")
		authRoutes.Post("/logout", adminAuthHandler.Logout)
		authRoutes.Post("/logout-all", adminAuthHandler.LogoutAll)
		authRoutes.Get("/me", adminAuthHandler.GetMe)
		authRoutes.Get("/sessions", adminAuthHandler.GetSessions)
		authRoutes.Delete("/sessions/:sessionId", adminAuthHandler.RevokeSession)
		authRoutes.Post("/change-password", adminAuthHandler.ChangePassword)
		authRoutes.Get("/login-history", adminAuthHandler.GetLoginHistory)

		// ========== Dashboard ==========
		dashboard := protected.Group("/dashboard")
		dashboard.Get("/summary", adminDashboardHandler.GetSummary)
		dashboard.Get("/activity", adminDashboardHandler.GetActivity)
		dashboard.Get("/charts", adminDashboardHandler.GetCharts)
		dashboard.Get("/stats", adminDashboardHandler.GetStats)
		dashboard.Get("/notifications", adminDashboardHandler.GetNotifications)
		dashboard.Put("/notifications/:id/read", adminDashboardHandler.MarkNotificationRead)

		// ========== Templates ==========
		templates := protected.Group("/templates")
		templates.Get("/", adminTemplateHandler.GetAll)
		templates.Get("/:id", adminTemplateHandler.GetByID)
		templates.Post("/", adminTemplateHandler.Create)
		templates.Put("/:id", adminTemplateHandler.Update)
		templates.Delete("/:id", adminTemplateHandler.Delete)
		templates.Get("/analytics", func(c *fiber.Ctx) error {
			logger.Info("Analytics route hit!")
			return adminTemplateHandler.GetAnalytics(c)
		})

		// ========== Categories ==========
		categories := protected.Group("/categories")
		// Template categories
		categories.Get("/templates", adminCategoryHandler.GetTemplateCategories)
		categories.Post("/templates", adminCategoryHandler.CreateTemplateCategory)
		categories.Put("/templates/:id", adminCategoryHandler.UpdateTemplateCategory)
		categories.Delete("/templates/:id", adminCategoryHandler.DeleteTemplateCategory)
		// Blog categories
		categories.Get("/blog", adminCategoryHandler.GetBlogCategories)
		categories.Post("/blog", adminCategoryHandler.CreateBlogCategory)
		categories.Put("/blog/:id", adminCategoryHandler.UpdateBlogCategory)
		categories.Delete("/blog/:id", adminCategoryHandler.DeleteBlogCategory)

		// ========== Orders ==========
		orders := protected.Group("/orders")
		orders.Get("/", adminOrderHandler.GetAll)
		orders.Get("/pending", adminOrderHandler.GetPending)
		orders.Get("/:id", adminOrderHandler.GetByID)
		orders.Post("/:id/approve", adminOrderHandler.Approve)
		orders.Post("/:id/reject", adminOrderHandler.Reject)
		orders.Get("/analytics/revenue", adminOrderHandler.GetRevenueAnalytics)

		// ========== Blog ==========
		blog := protected.Group("/blog")
		blog.Get("/posts", adminBlogHandler.GetAll)
		blog.Get("/posts/:id", adminBlogHandler.GetByID)
		blog.Post("/posts", adminBlogHandler.Create)
		blog.Put("/posts/:id", adminBlogHandler.Update)
		blog.Delete("/posts/:id", adminBlogHandler.Delete)
		blog.Get("/analytics", adminBlogHandler.GetAnalytics)

		// ========== Newsletter ==========
		newsletter := protected.Group("/newsletter")
		newsletter.Get("/subscribers", adminNewsletterHandler.GetAll)
		newsletter.Post("/subscribers", adminNewsletterHandler.Add)
		newsletter.Delete("/subscribers/:id", adminNewsletterHandler.Delete)
		newsletter.Get("/analytics", adminNewsletterHandler.GetAnalytics)
		newsletter.Post("/export", adminNewsletterHandler.Export)

		// ========== Contacts ==========
		contacts := protected.Group("/contacts")
		contacts.Get("/", adminContactHandler.GetAll)
		contacts.Get("/:id", adminContactHandler.GetByID)
		contacts.Put("/:id", adminContactHandler.Update)
		contacts.Post("/:id/reply", adminContactHandler.Reply)
		contacts.Delete("/:id", adminContactHandler.Delete)
		contacts.Get("/analytics", adminContactHandler.GetAnalytics)

		// ========== Tests ==========
		tests := protected.Group("/tests")
		// Questions
		tests.Get("/questions", adminTestHandler.GetAllQuestions)
		tests.Get("/questions/:id", adminTestHandler.GetQuestionByID)
		tests.Post("/questions", adminTestHandler.CreateQuestion)
		tests.Put("/questions/:id", adminTestHandler.UpdateQuestion)
		tests.Delete("/questions/:id", adminTestHandler.DeleteQuestion)
		// Submissions
		tests.Get("/submissions", adminTestHandler.GetAllSubmissions)
		tests.Get("/analytics", adminTestHandler.GetAnalytics)
		tests.Post("/export", adminTestHandler.Export)

		// ========== Calculators ==========
		calculators := protected.Group("/calculators")
		calculators.Get("/results", adminCalculatorHandler.GetAll)
		calculators.Get("/analytics", adminCalculatorHandler.GetAnalytics)

		// ========== Admin Users ==========
		admins := protected.Group("/users")
		admins.Get("/", adminUserHandler.GetAll)
		admins.Get("/:id", adminUserHandler.GetByID)
		admins.Post("/create", adminUserHandler.Create)
		admins.Put("/:id", adminUserHandler.Update)
		admins.Delete("/:id", adminUserHandler.Delete)

		// ========== Search (Global Admin Search) ==========
		protected.Get("/search", adminDashboardHandler.GlobalSearch)

		// ========== Settings ==========
		settings := protected.Group("/settings")
		settings.Get("/", adminDashboardHandler.GetSettings)
		settings.Put("/", adminDashboardHandler.UpdateSettings)
	}

	// 404 handler
	app.Use(func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Route not found",
			"code":    "NOT_FOUND",
			"path":    c.Path(),
		})
	})

	// Start server
	go func() {
		addr := fmt.Sprintf(":%d", cfg.Server.Port)
		logger.Info("🚀 Server starting",
			zap.String("address", addr),
			zap.String("env", cfg.Server.Environment),
		)
		logger.Info("📚 API Documentation: http://localhost:8000/api/v1/docs")
		logger.Info("❤️  Health Check: http://localhost:8000/health")

		if err := app.Listen(addr); err != nil {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	logger.Info("⚠️  Shutting down server...")
	if err := app.Shutdown(); err != nil {
		logger.Error("Server shutdown error", zap.Error(err))
	}

	logger.Info("✅ Server stopped gracefully")
}
