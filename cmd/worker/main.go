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
	"github.com/merraki/merraki-backend/internal/repository/redis"
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

	logger.Info("Starting Merraki Worker",
		zap.String("env", cfg.Server.Environment),
	)

	// Initialize database
	db, err := postgres.NewDatabase(cfg)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Initialize Redis
	_, err = redis.NewRedisClient(cfg)
	if err != nil {
		logger.Fatal("Failed to connect to Redis", zap.Error(err))
	}

	// Initialize repositories
	sessionRepo := postgres.NewSessionRepository(db)

	// Create context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start background jobs
	go cleanupExpiredSessions(ctx, sessionRepo)
	go processEmailQueue(ctx, cfg)
	go generateScheduledReports(ctx, cfg)

	logger.Info("Worker started successfully")

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down worker...")
	cancel()
	time.Sleep(2 * time.Second)
	logger.Info("Worker stopped")
}

func cleanupExpiredSessions(ctx context.Context, sessionRepo *postgres.SessionRepository) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			logger.Info("Cleaning up expired sessions...")
			if err := sessionRepo.CleanExpired(ctx); err != nil {
				logger.Error("Failed to cleanup sessions", zap.Error(err))
			} else {
				logger.Info("Expired sessions cleaned up")
			}
		}
	}
}

func processEmailQueue(ctx context.Context, cfg *config.Config) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// TODO: Process pending emails from queue
			logger.Debug("Processing email queue...")
		}
	}
}

func generateScheduledReports(ctx context.Context, cfg *config.Config) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// TODO: Generate daily/weekly reports
			logger.Info("Generating scheduled reports...")
		}
	}
}
