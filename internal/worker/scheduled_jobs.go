package worker

import (
	"context"
	"time"

	"github.com/merraki/merraki-backend/internal/domain"
	"github.com/merraki/merraki-backend/internal/pkg/logger"
	"github.com/merraki/merraki-backend/internal/repository"
	"go.uber.org/zap"
)

// ============================================================================
// SCHEDULED JOBS - Periodic maintenance tasks
// ============================================================================

type ScheduledJobRunner struct {
	jobRepo repository.BackgroundJobRepository
	ticker  *time.Ticker
	done    chan struct{}
}

func NewScheduledJobRunner(jobRepo repository.BackgroundJobRepository) *ScheduledJobRunner {
	return &ScheduledJobRunner{
		jobRepo: jobRepo,
		ticker:  time.NewTicker(1 * time.Hour), // Run every hour
		done:    make(chan struct{}),
	}
}

func (s *ScheduledJobRunner) Start(ctx context.Context) error {
	logger.Info("Starting scheduled job runner")

	// Run once immediately
	s.scheduleJobs(ctx)

	for {
		select {
		case <-ctx.Done():
			s.ticker.Stop()
			return nil

		case <-s.done:
			s.ticker.Stop()
			return nil

		case <-s.ticker.C:
			s.scheduleJobs(ctx)
		}
	}
}

func (s *ScheduledJobRunner) Stop() {
	close(s.done)
}

func (s *ScheduledJobRunner) scheduleJobs(ctx context.Context) {
	logger.Info("Scheduling periodic jobs")

	// Schedule cleanup jobs
	s.scheduleCleanupExpiredTokens(ctx)
	s.scheduleCleanupIdempotencyKeys(ctx)
}

func (s *ScheduledJobRunner) scheduleCleanupExpiredTokens(ctx context.Context) {
	job := &domain.BackgroundJob{
		JobType:     "cleanup_expired_tokens",
		Payload:     make(domain.JSONMap),
		Status:      domain.JobStatusPending,
		MaxRetries:  3,
		ScheduledAt: time.Now(),
		Priority:    0,
	}

	if err := s.jobRepo.Create(ctx, job); err != nil {
		logger.Error("Failed to schedule cleanup_expired_tokens job", zap.Error(err))
	}
}

func (s *ScheduledJobRunner) scheduleCleanupIdempotencyKeys(ctx context.Context) {
	job := &domain.BackgroundJob{
		JobType:     "cleanup_idempotency_keys",
		Payload:     make(domain.JSONMap),
		Status:      domain.JobStatusPending,
		MaxRetries:  3,
		ScheduledAt: time.Now(),
		Priority:    0,
	}

	if err := s.jobRepo.Create(ctx, job); err != nil {
		logger.Error("Failed to schedule cleanup_idempotency_keys job", zap.Error(err))
	}
}