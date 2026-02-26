package domain

import "time"

type TestQuestion struct {
	ID           int64     `db:"id" json:"id"`
	QuestionText string    `db:"question_text" json:"question_text"`
	QuestionType string    `db:"question_type" json:"question_type"`
	Options      JSONMap   `db:"options" json:"options"`
	Category     string    `db:"category" json:"category"`
	Weight       int       `db:"weight" json:"weight"`
	OrderNumber  int       `db:"order_number" json:"order_number"`
	IsActive     bool      `db:"is_active" json:"is_active"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

type TestSubmission struct {
	ID              int64     `db:"id" json:"id"`
	TestNumber      string    `db:"test_number" json:"test_number"`
	Email           string    `db:"email" json:"email"`
	Name            *string   `db:"name" json:"name,omitempty"`
	Responses       JSONMap   `db:"responses" json:"responses"`
	PersonalityType *string   `db:"personality_type" json:"personality_type,omitempty"`
	RiskAppetite    *string   `db:"risk_appetite" json:"risk_appetite,omitempty"`
	Scores          JSONMap   `db:"scores" json:"scores,omitempty"`
	ReportURL       *string   `db:"report_url" json:"report_url,omitempty"`
	ReportSent      bool      `db:"report_sent" json:"report_sent"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
}