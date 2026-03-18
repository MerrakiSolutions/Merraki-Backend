package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/merraki/merraki-backend/internal/domain"
)

type BackgroundJobRepository struct {
	db *sqlx.DB
}

func NewBackgroundJobRepository(db *sqlx.DB) *BackgroundJobRepository {
	return &BackgroundJobRepository{db: db}
}

func (r *BackgroundJobRepository) Create(ctx context.Context, job *domain.BackgroundJob) error {
	query := `
		INSERT INTO background_jobs (
			job_type, job_id, payload, status, max_retries, scheduled_at, priority
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`

	return r.db.QueryRowContext(
		ctx, query,
		job.JobType, job.JobID, job.Payload, job.Status,
		job.MaxRetries, job.ScheduledAt, job.Priority,
	).Scan(&job.ID, &job.CreatedAt, &job.UpdatedAt)
}

func (r *BackgroundJobRepository) FindByID(ctx context.Context, id int64) (*domain.BackgroundJob, error) {
	var job domain.BackgroundJob
	query := `SELECT * FROM background_jobs WHERE id = $1`

	err := r.db.GetContext(ctx, &job, query, id)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return &job, err
}

func (r *BackgroundJobRepository) GetPending(ctx context.Context, limit int) ([]*domain.BackgroundJob, error) {
	var jobs []*domain.BackgroundJob
	query := `
		SELECT * FROM background_jobs 
		WHERE status IN ('pending', 'retrying')
		AND scheduled_at <= CURRENT_TIMESTAMP
		AND (locked_at IS NULL OR lock_expires_at < CURRENT_TIMESTAMP)
		ORDER BY priority DESC, created_at ASC
		LIMIT $1
	`
	err := r.db.SelectContext(ctx, &jobs, query, limit)
	return jobs, err
}

func (r *BackgroundJobRepository) AcquireLock(ctx context.Context, id int64, workerID string, lockDuration int) (bool, error) {
	lockExpiry := time.Now().Add(time.Duration(lockDuration) * time.Second)

	query := `
		UPDATE background_jobs 
		SET locked_at = CURRENT_TIMESTAMP, locked_by = $1, lock_expires_at = $2, status = 'processing'
		WHERE id = $3 
		AND (locked_at IS NULL OR lock_expires_at < CURRENT_TIMESTAMP)
	`

	result, err := r.db.ExecContext(ctx, query, workerID, lockExpiry, id)
	if err != nil {
		return false, err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return false, err
	}

	return rows > 0, nil
}

func (r *BackgroundJobRepository) UpdateStatus(ctx context.Context, id int64, status domain.JobStatus) error {
	query := `UPDATE background_jobs SET status = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, status, id)
	return err
}

func (r *BackgroundJobRepository) MarkAsCompleted(ctx context.Context, id int64) error {
	query := `
		UPDATE background_jobs 
		SET status = 'completed', completed_at = CURRENT_TIMESTAMP, locked_at = NULL, locked_by = NULL
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *BackgroundJobRepository) MarkAsFailed(ctx context.Context, id int64, errorMsg string) error {
	query := `
		UPDATE background_jobs 
		SET status = 'failed', failed_at = CURRENT_TIMESTAMP, last_error = $1, locked_at = NULL, locked_by = NULL
		WHERE id = $2
	`
	_, err := r.db.ExecContext(ctx, query, errorMsg, id)
	return err
}

func (r *BackgroundJobRepository) IncrementRetryCount(ctx context.Context, id int64) error {
	query := `
		UPDATE background_jobs 
		SET retry_count = retry_count + 1,
			status = CASE 
				WHEN retry_count + 1 >= max_retries THEN 'failed'::job_status 
				ELSE 'retrying'::job_status 
			END,
			next_retry_at = CURRENT_TIMESTAMP + (POWER(2, retry_count + 1) * INTERVAL '1 minute'),
			locked_at = NULL,
			locked_by = NULL
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *BackgroundJobRepository) GetJobsByType(ctx context.Context, jobType string, limit int) ([]*domain.BackgroundJob, error) {
	var jobs []*domain.BackgroundJob
	query := `
		SELECT * FROM background_jobs 
		WHERE job_type = $1 
		ORDER BY created_at DESC 
		LIMIT $2
	`
	err := r.db.SelectContext(ctx, &jobs, query, jobType, limit)
	return jobs, err
}