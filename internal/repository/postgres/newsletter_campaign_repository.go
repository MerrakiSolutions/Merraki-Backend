package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/merraki/merraki-backend/internal/domain"
)

type NewsletterCampaignRepository struct {
	db *Database
}

func NewNewsletterCampaignRepository(db *Database) *NewsletterCampaignRepository {
	return &NewsletterCampaignRepository{db: db}
}

func (r *NewsletterCampaignRepository) Create(ctx context.Context, campaign *domain.NewsletterCampaign) error {
	query := `
		INSERT INTO newsletter_campaigns 
		(subject, slug, content, plain_text, from_name, from_email, reply_to, 
		 preview_text, status, scheduled_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, updated_at`

	return r.db.DB.QueryRowContext(
		ctx, query,
		campaign.Subject, campaign.Slug, campaign.Content, campaign.PlainText,
		campaign.FromName, campaign.FromEmail, campaign.ReplyTo, campaign.PreviewText,
		campaign.Status, campaign.ScheduledAt, campaign.CreatedBy,
	).Scan(&campaign.ID, &campaign.CreatedAt, &campaign.UpdatedAt)
}

func (r *NewsletterCampaignRepository) FindByID(ctx context.Context, id int64) (*domain.NewsletterCampaign, error) {
	var campaign domain.NewsletterCampaign
	query := `SELECT * FROM newsletter_campaigns WHERE id = $1`

	err := r.db.DB.GetContext(ctx, &campaign, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &campaign, err
}

func (r *NewsletterCampaignRepository) FindBySlug(ctx context.Context, slug string) (*domain.NewsletterCampaign, error) {
	var campaign domain.NewsletterCampaign
	query := `SELECT * FROM newsletter_campaigns WHERE slug = $1`

	err := r.db.DB.GetContext(ctx, &campaign, query, slug)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &campaign, err
}

func (r *NewsletterCampaignRepository) GetAll(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*domain.NewsletterCampaign, int, error) {
	var campaigns []*domain.NewsletterCampaign
	
	query := `SELECT * FROM newsletter_campaigns WHERE 1=1`
	countQuery := `SELECT COUNT(*) FROM newsletter_campaigns WHERE 1=1`
	args := []interface{}{}
	argCount := 1

	if status, ok := filters["status"].(string); ok && status != "" {
		query += fmt.Sprintf(" AND status = $%d", argCount)
		countQuery += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, status)
		argCount++
	}

	query += " ORDER BY created_at DESC"
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argCount, argCount+1)

	var total int
	if err := r.db.DB.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	if err := r.db.DB.SelectContext(ctx, &campaigns, query, args...); err != nil {
		return nil, 0, err
	}

	return campaigns, total, nil
}

func (r *NewsletterCampaignRepository) Update(ctx context.Context, campaign *domain.NewsletterCampaign) error {
	query := `
		UPDATE newsletter_campaigns 
		SET subject = $1, content = $2, plain_text = $3, from_name = $4,
		    from_email = $5, reply_to = $6, preview_text = $7, status = $8,
		    scheduled_at = $9, updated_at = NOW()
		WHERE id = $10
		RETURNING updated_at`

	return r.db.DB.QueryRowContext(
		ctx, query,
		campaign.Subject, campaign.Content, campaign.PlainText, campaign.FromName,
		campaign.FromEmail, campaign.ReplyTo, campaign.PreviewText, campaign.Status,
		campaign.ScheduledAt, campaign.ID,
	).Scan(&campaign.UpdatedAt)
}

func (r *NewsletterCampaignRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM newsletter_campaigns WHERE id = $1`
	_, err := r.db.DB.ExecContext(ctx, query, id)
	return err
}

func (r *NewsletterCampaignRepository) UpdateStats(ctx context.Context, campaignID int64) error {
	query := `
		UPDATE newsletter_campaigns 
		SET total_recipients = (SELECT COUNT(*) FROM newsletter_campaign_recipients WHERE campaign_id = $1),
		    total_sent = (SELECT COUNT(*) FROM newsletter_campaign_recipients WHERE campaign_id = $1 AND status = 'sent'),
		    total_failed = (SELECT COUNT(*) FROM newsletter_campaign_recipients WHERE campaign_id = $1 AND status = 'failed'),
		    total_opened = (SELECT COUNT(*) FROM newsletter_campaign_recipients WHERE campaign_id = $1 AND opened_at IS NOT NULL),
		    total_clicked = (SELECT COUNT(*) FROM newsletter_campaign_recipients WHERE campaign_id = $1 AND clicked_at IS NOT NULL)
		WHERE id = $1`

	_, err := r.db.DB.ExecContext(ctx, query, campaignID)
	return err
}

// Recipient tracking
func (r *NewsletterCampaignRepository) AddRecipients(ctx context.Context, campaignID int64, subscriberIDs []int64) error {
	if len(subscriberIDs) == 0 {
		return nil
	}

	query := `
		INSERT INTO newsletter_campaign_recipients (campaign_id, subscriber_id, status)
		VALUES ($1, $2, 'pending')
		ON CONFLICT (campaign_id, subscriber_id) DO NOTHING`

	for _, subscriberID := range subscriberIDs {
		if _, err := r.db.DB.ExecContext(ctx, query, campaignID, subscriberID); err != nil {
			return err
		}
	}

	return nil
}

func (r *NewsletterCampaignRepository) GetRecipients(ctx context.Context, campaignID int64) ([]*domain.NewsletterCampaignRecipient, error) {
	var recipients []*domain.NewsletterCampaignRecipient
	query := `SELECT * FROM newsletter_campaign_recipients WHERE campaign_id = $1 ORDER BY created_at DESC`

	err := r.db.DB.SelectContext(ctx, &recipients, query, campaignID)
	return recipients, err
}