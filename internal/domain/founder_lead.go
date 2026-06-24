package domain

import (
	"time"
)

// ============================================================================
// FOUNDER'S TEST LEAD - Domain model
// ============================================================================

type LeadStatus string

const (
	LeadStatusNew       LeadStatus = "new"
	LeadStatusContacted LeadStatus = "contacted"
	LeadStatusQualified LeadStatus = "qualified"
	LeadStatusConverted LeadStatus = "converted"
	LeadStatusRejected  LeadStatus = "rejected"
)

type SectionScore struct {
	Dimension  string `json:"dimension"`
	Label      string `json:"label"`
	Score      int    `json:"score"`
	Max        int    `json:"max"`
	Percentage int    `json:"percentage"`
}

type FounderLead struct {
	ID                     int64            `json:"id" db:"id"`
	Name                   string           `json:"name" db:"name"`
	Email                  string           `json:"email" db:"email"`
	Company                string           `json:"company" db:"company"`
	Role                   string           `json:"role" db:"role"`
	IPAddress              string           `json:"ip_address" db:"ip_address"`
	TotalScore             int              `json:"total_score" db:"total_score"`
	TotalMax               int              `json:"total_max" db:"total_max"`
	PersonalityType        string           `json:"personality_type" db:"personality_type"`
	PersonalityTitle       string           `json:"personality_title" db:"personality_title"`
	PersonalityBadge       string           `json:"personality_badge" db:"personality_badge"`
	PersonalityColor       string           `json:"personality_color" db:"personality_color"`
	PersonalityDescription string           `json:"personality_description" db:"personality_description"`
	SectionScores          []SectionScore   `json:"section_scores" db:"section_scores"`
	Status                 LeadStatus       `json:"status" db:"status"`
	Notes                  string           `json:"notes" db:"notes"`
	SubmittedAt            time.Time        `json:"submitted_at" db:"submitted_at"`
	UpdatedAt              time.Time        `json:"updated_at" db:"updated_at"`
}

// ============================================================================
// REQUEST TYPES
// ============================================================================

type SubmitFounderLeadRequest struct {
	Name                   string         `json:"name" validate:"required,min=2"`
	Email                  string         `json:"email" validate:"required,email"`
	Company                string         `json:"company"`
	Role                   string         `json:"role"`
	TotalScore             int            `json:"total_score" validate:"required"`
	TotalMax               int            `json:"total_max" validate:"required"`
	PersonalityType        string         `json:"personality_type" validate:"required"`
	PersonalityTitle       string         `json:"personality_title" validate:"required"`
	PersonalityBadge       string         `json:"personality_badge" validate:"required"`
	PersonalityColor       string         `json:"personality_color" validate:"required"`
	PersonalityDescription string         `json:"personality_description" validate:"required"`
	SectionScores          []SectionScore `json:"section_scores" validate:"required"`
}

type PatchLeadStatusRequest struct {
	Status LeadStatus `json:"status" validate:"required"`
}

type SendLeadEmailRequest struct {
	Subject string `json:"subject" validate:"required"`
	Message string `json:"message" validate:"required"`
}

type FounderLeadStats struct {
	Total          int     `json:"total"`
	NewThisWeek    int     `json:"new_this_week"`
	Contacted      int     `json:"contacted"`
	Qualified      int     `json:"qualified"`
	Converted      int     `json:"converted"`
	AvgScore       float64 `json:"avg_score"`
	ConversionRate float64 `json:"conversion_rate"`
}