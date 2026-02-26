package postgres

import (
	"context"
	"database/sql"

	"github.com/merraki/merraki-backend/internal/domain"
)

type SessionRepository struct {
	db *Database
}

func NewSessionRepository(db *Database) *SessionRepository {
	return &SessionRepository{db: db}
}

func (r *SessionRepository) Create(ctx context.Context, session *domain.AdminSession) error {
	query := `
		INSERT INTO admin_sessions 
		(admin_id, token_hash, device_name, ip_address, user_agent, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`

	return r.db.DB.QueryRowContext(
		ctx, query,
		session.AdminID, session.TokenHash, session.DeviceName,
		session.IPAddress, session.UserAgent, session.ExpiresAt,
	).Scan(&session.ID, &session.CreatedAt)
}

func (r *SessionRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*domain.AdminSession, error) {
	var session domain.AdminSession
	query := `
		SELECT * FROM admin_sessions 
		WHERE token_hash = $1 
		  AND is_active = true 
		  AND expires_at > NOW()`

	err := r.db.DB.GetContext(ctx, &session, query, tokenHash)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &session, err
}

func (r *SessionRepository) UpdateLastUsed(ctx context.Context, id int64) error {
	query := `UPDATE admin_sessions SET last_used_at = NOW() WHERE id = $1`
	_, err := r.db.DB.ExecContext(ctx, query, id)
	return err
}

func (r *SessionRepository) Revoke(ctx context.Context, id int64) error {
	query := `UPDATE admin_sessions SET is_active = false WHERE id = $1`
	_, err := r.db.DB.ExecContext(ctx, query, id)
	return err
}

func (r *SessionRepository) RevokeAllForAdmin(ctx context.Context, adminID int64) error {
	query := `UPDATE admin_sessions SET is_active = false WHERE admin_id = $1`
	_, err := r.db.DB.ExecContext(ctx, query, adminID)
	return err
}

func (r *SessionRepository) GetActiveSessions(ctx context.Context, adminID int64) ([]*domain.AdminSession, error) {
	var sessions []*domain.AdminSession
	query := `
		SELECT * FROM admin_sessions 
		WHERE admin_id = $1 
		  AND is_active = true 
		  AND expires_at > NOW()
		ORDER BY last_used_at DESC`

	err := r.db.DB.SelectContext(ctx, &sessions, query, adminID)
	return sessions, err
}

func (r *SessionRepository) CountActiveSessions(ctx context.Context, adminID int64) (int, error) {
	var count int
	query := `
		SELECT COUNT(*) FROM admin_sessions 
		WHERE admin_id = $1 
		  AND is_active = true 
		  AND expires_at > NOW()`

	err := r.db.DB.GetContext(ctx, &count, query, adminID)
	return count, err
}

func (r *SessionRepository) CleanExpired(ctx context.Context) error {
	query := `DELETE FROM admin_sessions WHERE expires_at < NOW() - INTERVAL '7 days'`
	_, err := r.db.DB.ExecContext(ctx, query)
	return err
}