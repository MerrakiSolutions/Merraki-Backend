package domain

import (
	"time"

	"github.com/lib/pq"
)

// BlogAuthor represents a blog post author
type BlogAuthor struct {
	ID          int64      `db:"id" json:"id"`
	AdminID     *int64     `db:"admin_id" json:"admin_id,omitempty"`
	Name        string     `db:"name" json:"name"`
	Slug        string     `db:"slug" json:"slug"`
	Email       *string    `db:"email" json:"email,omitempty"`
	Bio         *string    `db:"bio" json:"bio,omitempty"`
	AvatarURL   *string    `db:"avatar_url" json:"avatar_url,omitempty"`
	SocialLinks JSONMap    `db:"social_links" json:"social_links,omitempty"`
	IsActive    bool       `db:"is_active" json:"is_active"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at" json:"updated_at"`
}

// BlogCategory represents a blog category
type BlogCategory struct {
	ID           int64     `db:"id" json:"id"`
	Name         string    `db:"name" json:"name"`
	Slug         string    `db:"slug" json:"slug"`
	Description  *string   `db:"description" json:"description,omitempty"`
	ParentID     *int64    `db:"parent_id" json:"parent_id,omitempty"`
	DisplayOrder int       `db:"display_order" json:"display_order"`
	IsActive     bool      `db:"is_active" json:"is_active"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

// BlogPost represents a blog post
type BlogPost struct {
	ID                 int64          `db:"id" json:"id"`
	Title              string         `db:"title" json:"title"`
	Slug               string         `db:"slug" json:"slug"`
	Excerpt            *string        `db:"excerpt" json:"excerpt,omitempty"`
	Content            string         `db:"content" json:"content"`
	FeaturedImageURL   *string        `db:"featured_image_url" json:"featured_image_url,omitempty"`
	AuthorID           *int64         `db:"author_id" json:"author_id,omitempty"`
	CategoryID         *int64         `db:"category_id" json:"category_id,omitempty"`
	Tags               pq.StringArray `db:"tags" json:"tags"`
	MetaTitle          *string        `db:"meta_title" json:"meta_title,omitempty"`
	MetaDescription    *string        `db:"meta_description" json:"meta_description,omitempty"`
	MetaKeywords       pq.StringArray `db:"meta_keywords" json:"meta_keywords,omitempty"`
	Status             string         `db:"status" json:"status"`
	IsFeatured         bool           `db:"is_featured" json:"is_featured"`
	ViewsCount         int            `db:"views_count" json:"views_count"`
	ReadingTimeMinutes *int           `db:"reading_time_minutes" json:"reading_time_minutes,omitempty"`
	PublishedAt        *time.Time     `db:"published_at" json:"published_at,omitempty"`
	CreatedAt          time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt          time.Time      `db:"updated_at" json:"updated_at"`
}

// BlogPostWithRelations - For API responses with joined data
type BlogPostWithRelations struct {
	BlogPost
	Author   *BlogAuthor   `json:"author,omitempty"`
	Category *BlogCategory `json:"category,omitempty"`
}