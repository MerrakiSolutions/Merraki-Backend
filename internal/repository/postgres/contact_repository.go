package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/merraki/merraki-backend/internal/domain"
)

type ContactRepository struct {
	db *Database
}

func NewContactRepository(db *Database) *ContactRepository {
	return &ContactRepository{db: db}
}

func (r *ContactRepository) Create(ctx context.Context, contact *domain.Contact) error {
	query := `
		INSERT INTO contacts (name, email, phone, subject, message, status, ip_address)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`

	return r.db.DB.QueryRowContext(
		ctx, query,
		contact.Name, contact.Email, contact.Phone, contact.Subject,
		contact.Message, contact.Status, contact.IPAddress,
	).Scan(&contact.ID, &contact.CreatedAt)
}

func (r *ContactRepository) FindByID(ctx context.Context, id int64) (*domain.Contact, error) {
	var contact domain.Contact
	query := `SELECT * FROM contacts WHERE id = $1`

	err := r.db.DB.GetContext(ctx, &contact, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &contact, err
}

func (r *ContactRepository) GetAll(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*domain.Contact, int, error) {
	var contacts []*domain.Contact
	
	query := `SELECT * FROM contacts WHERE 1=1`
	countQuery := `SELECT COUNT(*) FROM contacts WHERE 1=1`
	args := []interface{}{}
	argCount := 1

	if status, ok := filters["status"].(string); ok && status != "" {
		query += fmt.Sprintf(" AND status = $%d", argCount)
		countQuery += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, status)
		argCount++
	}

	if search, ok := filters["search"].(string); ok && search != "" {
		query += fmt.Sprintf(" AND (email ILIKE $%d OR name ILIKE $%d OR subject ILIKE $%d)", argCount, argCount, argCount)
		countQuery += fmt.Sprintf(" AND (email ILIKE $%d OR name ILIKE $%d OR subject ILIKE $%d)", argCount, argCount, argCount)
		args = append(args, "%"+search+"%")
		argCount++
	}

	query += " ORDER BY created_at DESC"
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argCount, argCount+1)

	var total int
	if err := r.db.DB.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	if err := r.db.DB.SelectContext(ctx, &contacts, query, args...); err != nil {
		return nil, 0, err
	}

	return contacts, total, nil
}

func (r *ContactRepository) Update(ctx context.Context, contact *domain.Contact) error {
	query := `
		UPDATE contacts 
		SET status = $1, replied_by = $2, replied_at = $3, reply_notes = $4
		WHERE id = $5`

	_, err := r.db.DB.ExecContext(
		ctx, query,
		contact.Status, contact.RepliedBy, contact.RepliedAt, contact.ReplyNotes, contact.ID,
	)
	return err
}

func (r *ContactRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM contacts WHERE id = $1`
	_, err := r.db.DB.ExecContext(ctx, query, id)
	return err
}

func (r *ContactRepository) GetAnalytics(ctx context.Context) (map[string]interface{}, error) {
	analytics := make(map[string]interface{})

	var counts struct {
		Total      int `db:"total"`
		New        int `db:"new"`
		InProgress int `db:"in_progress"`
		Replied    int `db:"replied"`
		Closed     int `db:"closed"`
	}
	countsQuery := `
		SELECT 
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'new') as new,
			COUNT(*) FILTER (WHERE status = 'in_progress') as in_progress,
			COUNT(*) FILTER (WHERE status = 'replied') as replied,
			COUNT(*) FILTER (WHERE status = 'closed') as closed
		FROM contacts`
	
	if err := r.db.DB.GetContext(ctx, &counts, countsQuery); err != nil {
		return nil, err
	}

	analytics["total_contacts"] = counts.Total
	analytics["by_status"] = map[string]int{
		"new":         counts.New,
		"in_progress": counts.InProgress,
		"replied":     counts.Replied,
		"closed":      counts.Closed,
	}

	return analytics, nil
}