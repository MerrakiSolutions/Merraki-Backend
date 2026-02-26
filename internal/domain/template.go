package domain

import (
	"time"

	"github.com/lib/pq"
)

type Template struct {
	ID                  int64          `db:"id" json:"id"`
	Slug                string         `db:"slug" json:"slug"`
	Title               string         `db:"title" json:"title"`
	Description         *string        `db:"description" json:"description,omitempty"`
	DetailedDescription *string        `db:"detailed_description" json:"detailed_description,omitempty"`
	PriceINR            int            `db:"price_inr" json:"price_inr"`
	ThumbnailURL        *string        `db:"thumbnail_url" json:"thumbnail_url,omitempty"`
	PreviewURLs         pq.StringArray `db:"preview_urls" json:"preview_urls,omitempty"`
	FileURL             string         `db:"file_url" json:"file_url"`
	FileSizeBytes       *int64         `db:"file_size_bytes" json:"file_size_bytes,omitempty"`
	CategoryID          int64          `db:"category_id" json:"category_id"`
	Tags                pq.StringArray `db:"tags" json:"tags"`
	DownloadsCount      int            `db:"downloads_count" json:"downloads_count"`
	ViewsCount          int            `db:"views_count" json:"views_count"`
	Rating              float64        `db:"rating" json:"rating"`
	RatingCount         int            `db:"rating_count" json:"rating_count"`
	Status              string         `db:"status" json:"status"`
	IsFeatured          bool           `db:"is_featured" json:"is_featured"`
	MetaTitle           *string        `db:"meta_title" json:"meta_title,omitempty"`
	MetaDescription     *string        `db:"meta_description" json:"meta_description,omitempty"`
	CreatedBy           *int64         `db:"created_by" json:"created_by,omitempty"`
	CreatedAt           time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt           time.Time      `db:"updated_at" json:"updated_at"`
}