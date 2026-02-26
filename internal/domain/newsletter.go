package domain

import (
	"time"
)

type NewsletterSubscriber struct {
	ID             int64      `db:"id" json:"id"`
	Email          string     `db:"email" json:"email"`
	Name           *string    `db:"name" json:"name,omitempty"`
	Status         string     `db:"status" json:"status"`
	Source         string     `db:"source" json:"source"`
	IPAddress      *string    `db:"ip_address" json:"ip_address,omitempty"`
	SubscribedAt   time.Time  `db:"subscribed_at" json:"subscribed_at"`
	UnsubscribedAt *time.Time `db:"unsubscribed_at" json:"unsubscribed_at,omitempty"`
}

type NewsletterCampaign struct {
	ID              int64      `db:"id" json:"id"`
	Subject         string     `db:"subject" json:"subject"`
	Slug            string     `db:"slug" json:"slug"`
	Content         string     `db:"content" json:"content"`
	PlainText       *string    `db:"plain_text" json:"plain_text,omitempty"`
	FromName        string     `db:"from_name" json:"from_name"`
	FromEmail       string     `db:"from_email" json:"from_email"`
	ReplyTo         *string    `db:"reply_to" json:"reply_to,omitempty"`
	PreviewText     *string    `db:"preview_text" json:"preview_text,omitempty"`
	Status          string     `db:"status" json:"status"`
	ScheduledAt     *time.Time `db:"scheduled_at" json:"scheduled_at,omitempty"`
	SentAt          *time.Time `db:"sent_at" json:"sent_at,omitempty"`
	TotalRecipients int        `db:"total_recipients" json:"total_recipients"`
	TotalSent       int        `db:"total_sent" json:"total_sent"`
	TotalFailed     int        `db:"total_failed" json:"total_failed"`
	TotalOpened     int        `db:"total_opened" json:"total_opened"`
	TotalClicked    int        `db:"total_clicked" json:"total_clicked"`
	CreatedBy       *int64     `db:"created_by" json:"created_by,omitempty"`
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at" json:"updated_at"`
}

type NewsletterCampaignRecipient struct {
	ID           int64      `db:"id" json:"id"`
	CampaignID   int64      `db:"campaign_id" json:"campaign_id"`
	SubscriberID int64      `db:"subscriber_id" json:"subscriber_id"`
	Status       string     `db:"status" json:"status"`
	SentAt       *time.Time `db:"sent_at" json:"sent_at,omitempty"`
	OpenedAt     *time.Time `db:"opened_at" json:"opened_at,omitempty"`
	ClickedAt    *time.Time `db:"clicked_at" json:"clicked_at,omitempty"`
	ErrorMessage *string    `db:"error_message" json:"error_message,omitempty"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
}