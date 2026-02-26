package domain

import (
	"time"

	"github.com/lib/pq"
)

type BlogPost struct {
	ID                 int64          `db:"id" json:"id"`
	Slug               string         `db:"slug" json:"slug"`
	Title              string         `db:"title" json:"title"`
	Excerpt            *string        `db:"excerpt" json:"excerpt,omitempty"`
	Content            string         `db:"content" json:"content"`
	FeaturedImageURL   *string        `db:"featured_image_url" json:"featured_image_url,omitempty"`
	CategoryID         *int64         `db:"category_id" json:"category_id,omitempty"`
	Tags               pq.StringArray `db:"tags" json:"tags"`
	MetaTitle          *string        `db:"meta_title" json:"meta_title,omitempty"`
	MetaDescription    *string        `db:"meta_description" json:"meta_description,omitempty"`
	SEOKeywords        pq.StringArray `db:"seo_keywords" json:"seo_keywords,omitempty"`
	ViewsCount         int            `db:"views_count" json:"views_count"`
	ReadingTimeMinutes *int           `db:"reading_time_minutes" json:"reading_time_minutes,omitempty"`
	AuthorID           *int64         `db:"author_id" json:"author_id,omitempty"`
	Status             string         `db:"status" json:"status"`
	PublishedAt        *time.Time     `db:"published_at" json:"published_at,omitempty"`
	CreatedAt          time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt          time.Time      `db:"updated_at" json:"updated_at"`
}

type BlogCategory struct {
	ID           int64     `db:"id" json:"id"`
	Slug         string    `db:"slug" json:"slug"`
	Name         string    `db:"name" json:"name"`
	Description  *string   `db:"description" json:"description,omitempty"`
	DisplayOrder int       `db:"display_order" json:"display_order"`
	IsActive     bool      `db:"is_active" json:"is_active"`
	PostsCount   int       `db:"posts_count" json:"posts_count"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}