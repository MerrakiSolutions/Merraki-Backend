package domain

import "time"

type Contact struct {
	ID         int64      `db:"id" json:"id"`
	Name       string     `db:"name" json:"name"`
	Email      string     `db:"email" json:"email"`
	Phone      *string    `db:"phone" json:"phone,omitempty"`
	Subject    string     `db:"subject" json:"subject"`
	Message    string     `db:"message" json:"message"`
	Status     string     `db:"status" json:"status"`
	RepliedBy  *int64     `db:"replied_by" json:"replied_by,omitempty"`
	RepliedAt  *time.Time `db:"replied_at" json:"replied_at,omitempty"`
	ReplyNotes *string    `db:"reply_notes" json:"reply_notes,omitempty"`
	IPAddress  *string    `db:"ip_address" json:"ip_address,omitempty"`
	CreatedAt  time.Time  `db:"created_at" json:"created_at"`
}