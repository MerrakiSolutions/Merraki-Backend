package domain

import "time"

type TemplateCategory struct {
	ID              int64      `db:"id" json:"id"`
	Slug            string     `db:"slug" json:"slug"`
	Name            string     `db:"name" json:"name"`
	Description     *string    `db:"description" json:"description,omitempty"`
	IconName        *string    `db:"icon_name" json:"icon_name,omitempty"`
	DisplayOrder    int        `db:"display_order" json:"display_order"`
	ColorHex        *string    `db:"color_hex" json:"color_hex,omitempty"`
	MetaTitle       *string    `db:"meta_title" json:"meta_title,omitempty"`
	MetaDescription *string    `db:"meta_description" json:"meta_description,omitempty"`
	IsActive        bool       `db:"is_active" json:"is_active"`
	TemplatesCount  int        `db:"templates_count" json:"templates_count"`
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at" json:"updated_at"`
}

