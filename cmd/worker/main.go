package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/merraki/merraki-backend/internal/config"
	"github.com/merraki/merraki-backend/internal/pkg/logger"
	"github.com/merraki/merraki-backend/internal/repository/postgres"
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

	logger.Info("🔧 Starting Merraki Background Worker",
		zap.String("env", cfg.Server.Environment),
	)

	// ========================================================================
	// INITIALIZE DATABASE
	// ========================================================================
	db, err := postgres.NewDatabase(cfg)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	logger.Info("✅ Database connected")

	// ========================================================================
	// INITIALIZE REPOSITORIES
	// ========================================================================
	
	sessionRepo := postgres.NewSessionRepository(db)
	
	// Marketplace repos
	orderRepo := postgres.NewOrderRepository(db.DB)
	orderItemRepo := postgres.NewOrderItemRepository(db.DB)
	paymentRepo := postgres.NewPaymentRepository(db.DB)
	webhookRepo := postgres.NewPaymentWebhookRepository(db.DB)
	downloadTokenRepo := postgres.NewDownloadTokenRepository(db.DB)
	downloadRepo := postgres.NewDownloadRepository(db.DB)
	idempotencyRepo := postgres.NewIdempotencyKeyRepository(db.DB)
	templateRepo := postgres.NewTemplateRepository(db.DB)
	circuitBreakerRepo := postgres.NewCircuitBreakerRepository(db.DB)
	jobRepo := postgres.NewBackgroundJobRepository(db.DB)

	logger.Info("✅ Repositories initialized")

	// ========================================================================
	// INITIALIZE SERVICES
	// ========================================================================
	
	// Storage
	storageService, err := service.NewStorageService(cfg)
	if err != nil {
		logger.Fatal("Failed to initialize storage service", zap.Error(err))
	}

	// Email
	emailService := service.NewEmailService(cfg)

	// PDF
	pdfService := service.NewPDFService(storageService)

	// Payment
	paymentService := service.NewPaymentService(cfg, webhookRepo, circuitBreakerRepo)

	// Download token service
	downloadTokenService := service.NewDownloadTokenService(
		downloadTokenRepo,
		downloadRepo,
		orderRepo,
		orderItemRepo,
		templateRepo,
		storageService,
	)

	logger.Info("✅ Services initialized")

	// ========================================================================
	// INITIALIZE WORKERS
	// ========================================================================

	// Main job processor
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
		"worker-standalone-1",
	)

	// Scheduled job runner
	scheduledRunner := worker.NewScheduledJobRunner(jobRepo)

	// Create context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ========================================================================
	// START WORKERS
	// ========================================================================

	// Start job processor
	go func() {
		logger.Info("🚀 Starting job processor...")
		if err := jobProcessor.Start(ctx); err != nil {
			logger.Error("Job processor error", zap.Error(err))
		}
	}()

	// Start scheduled runner
	go func() {
		logger.Info("🚀 Starting scheduled job runner...")
		if err := scheduledRunner.Start(ctx); err != nil {
			logger.Error("Scheduled runner error", zap.Error(err))
		}
	}()

	// Start session cleanup
	go cleanupExpiredSessions(ctx, sessionRepo)

	logger.Info("✅ All workers started successfully")

	// ========================================================================
	// GRACEFUL SHUTDOWN
	// ========================================================================

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	logger.Info("⚠️  Shutting down workers...")

	// Cancel context to stop all workers
	cancel()

	// Stop workers gracefully
	jobProcessor.Stop()
	scheduledRunner.Stop()

	// Wait for cleanup
	time.Sleep(2 * time.Second)

	logger.Info("✅ Workers stopped gracefully")
}

// ============================================================================
// CLEANUP EXPIRED SESSIONS
// ============================================================================

func cleanupExpiredSessions(ctx context.Context, sessionRepo *postgres.SessionRepository) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Session cleanup stopped")
			return

		case <-ticker.C:
			logger.Info("Cleaning up expired sessions...")
			if err := sessionRepo.CleanExpired(ctx); err != nil {
				logger.Error("Failed to cleanup sessions", zap.Error(err))
			} else {
				logger.Info("✅ Expired sessions cleaned up")
			}
		}
	}
}