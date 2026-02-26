package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/merraki/merraki-backend/internal/domain"
)

type NewsletterRepository struct {
	db *Database
}

func (r *NewsletterRepository) FindByID(ctx context.Context, id int64) (any, any) {
	panic("unimplemented")
}

func NewNewsletterRepository(db *Database) *NewsletterRepository {
	return &NewsletterRepository{db: db}
}

func (r *NewsletterRepository) Create(ctx context.Context, subscriber *domain.NewsletterSubscriber) error {
	query := `
		INSERT INTO newsletter_subscribers (email, name, status, source, ip_address)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, subscribed_at`

	return r.db.DB.QueryRowContext(
		ctx, query,
		subscriber.Email, subscriber.Name, subscriber.Status,
		subscriber.Source, subscriber.IPAddress,
	).Scan(&subscriber.ID, &subscriber.SubscribedAt)
}

func (r *NewsletterRepository) FindByEmail(ctx context.Context, email string) (*domain.NewsletterSubscriber, error) {
	var subscriber domain.NewsletterSubscriber
	query := `SELECT * FROM newsletter_subscribers WHERE email = $1`

	err := r.db.DB.GetContext(ctx, &subscriber, query, email)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &subscriber, err
}

func (r *NewsletterRepository) GetAll(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*domain.NewsletterSubscriber, int, error) {
	var subscribers []*domain.NewsletterSubscriber

	query := `SELECT * FROM newsletter_subscribers WHERE 1=1`
	countQuery := `SELECT COUNT(*) FROM newsletter_subscribers WHERE 1=1`
	args := []interface{}{}
	argCount := 1

	if status, ok := filters["status"].(string); ok && status != "" {
		query += fmt.Sprintf(" AND status = $%d", argCount)
		countQuery += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, status)
		argCount++
	}

	if search, ok := filters["search"].(string); ok && search != "" {
		query += fmt.Sprintf(" AND (email ILIKE $%d OR name ILIKE $%d)", argCount, argCount)
		countQuery += fmt.Sprintf(" AND (email ILIKE $%d OR name ILIKE $%d)", argCount, argCount)
		args = append(args, "%"+search+"%")
		argCount++
	}

	query += " ORDER BY subscribed_at DESC"
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argCount, argCount+1)

	var total int
	if err := r.db.DB.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	if err := r.db.DB.SelectContext(ctx, &subscribers, query, args...); err != nil {
		return nil, 0, err
	}

	return subscribers, total, nil
}

func (r *NewsletterRepository) Unsubscribe(ctx context.Context, email string) error {
	query := `
		UPDATE newsletter_subscribers 
		SET status = 'unsubscribed', unsubscribed_at = NOW()
		WHERE email = $1`

	_, err := r.db.DB.ExecContext(ctx, query, email)
	return err
}



func (r *NewsletterRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM newsletter_subscribers WHERE id = $1`
	_, err := r.db.DB.ExecContext(ctx, query, id)
	return err
}

func (r *NewsletterRepository) GetAnalytics(ctx context.Context) (map[string]interface{}, error) {
	analytics := make(map[string]interface{})

	var counts struct {
		Total        int `db:"total"`
		Active       int `db:"active"`
		Unsubscribed int `db:"unsubscribed"`
	}
	countsQuery := `
		SELECT 
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'active') as active,
			COUNT(*) FILTER (WHERE status = 'unsubscribed') as unsubscribed
		FROM newsletter_subscribers`

	if err := r.db.DB.GetContext(ctx, &counts, countsQuery); err != nil {
		return nil, err
	}
	analytics["total_subscribers"] = counts.Total
	analytics["active_subscribers"] = counts.Active
	analytics["unsubscribed"] = counts.Unsubscribed

	var bySource []struct {
		Source string `db:"source"`
		Count  int    `db:"count"`
	}
	sourceQuery := `
		SELECT source, COUNT(*) as count 
		FROM newsletter_subscribers 
		WHERE status = 'active'
		GROUP BY source`

	if err := r.db.DB.SelectContext(ctx, &bySource, sourceQuery); err != nil {
		return nil, err
	}
	analytics["by_source"] = bySource

	return analytics, nil
}
