package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"
	"github.com/merraki/merraki-backend/internal/domain"
)

type BlogRepository struct {
	db *Database
}

func NewBlogRepository(db *Database) *BlogRepository {
	return &BlogRepository{db: db}
}

func (r *BlogRepository) Create(ctx context.Context, post *domain.BlogPost) error {
	query := `
		INSERT INTO blog_posts 
		(slug, title, excerpt, content, featured_image_url, category_id, tags,
		 meta_title, meta_description, seo_keywords, reading_time_minutes, 
		 author_id, status, published_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id, created_at, updated_at`

	err := r.db.DB.QueryRowContext(
		ctx, query,
		post.Slug, 
		post.Title, 
		post.Excerpt, 
		post.Content, 
		post.FeaturedImageURL,
		post.CategoryID, 
		pq.Array(post.Tags),  // Use pq.Array for TEXT[] columns
		post.MetaTitle, 
		post.MetaDescription,
		pq.Array(post.SEOKeywords),  // Use pq.Array for TEXT[] columns
		post.ReadingTimeMinutes, 
		post.AuthorID, 
		post.Status, 
		post.PublishedAt,
	).Scan(&post.ID, &post.CreatedAt, &post.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create blog post: %w", err)
	}

	return nil
}

func (r *BlogRepository) FindByID(ctx context.Context, id int64) (*domain.BlogPost, error) {
	var post domain.BlogPost
	query := `SELECT * FROM blog_posts WHERE id = $1`

	err := r.db.DB.GetContext(ctx, &post, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &post, nil
}

func (r *BlogRepository) FindBySlug(ctx context.Context, slug string) (*domain.BlogPost, error) {
	var post domain.BlogPost
	query := `SELECT * FROM blog_posts WHERE slug = $1`

	err := r.db.DB.GetContext(ctx, &post, query, slug)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &post, err
}

func (r *BlogRepository) GetAll(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*domain.BlogPost, int, error) {
	var posts []*domain.BlogPost
	
	query := `SELECT * FROM blog_posts WHERE 1=1`
	countQuery := `SELECT COUNT(*) FROM blog_posts WHERE 1=1`
	args := []interface{}{}
	argCount := 1

	if status, ok := filters["status"].(string); ok && status != "" {
		query += fmt.Sprintf(" AND status = $%d", argCount)
		countQuery += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, status)
		argCount++
	}

	if categoryID, ok := filters["category_id"].(int64); ok && categoryID > 0 {
		query += fmt.Sprintf(" AND category_id = $%d", argCount)
		countQuery += fmt.Sprintf(" AND category_id = $%d", argCount)
		args = append(args, categoryID)
		argCount++
	}

	if tag, ok := filters["tag"].(string); ok && tag != "" {
		query += fmt.Sprintf(" AND $%d = ANY(tags)", argCount)
		countQuery += fmt.Sprintf(" AND $%d = ANY(tags)", argCount)
		args = append(args, tag)
		argCount++
	}

	if search, ok := filters["search"].(string); ok && search != "" {
		query += fmt.Sprintf(" AND (title ILIKE $%d OR excerpt ILIKE $%d OR content ILIKE $%d)", argCount, argCount, argCount)
		countQuery += fmt.Sprintf(" AND (title ILIKE $%d OR excerpt ILIKE $%d OR content ILIKE $%d)", argCount, argCount, argCount)
		args = append(args, "%"+search+"%")
		argCount++
	}

	sort := "created_at DESC"
	if sortParam, ok := filters["sort"].(string); ok {
		switch sortParam {
		case "newest":
			sort = "published_at DESC NULLS LAST"
		case "oldest":
			sort = "published_at ASC NULLS LAST"
		case "popular":
			sort = "views_count DESC"
		}
	}

	query += " ORDER BY " + sort
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argCount, argCount+1)

	var total int
	if err := r.db.DB.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	if err := r.db.DB.SelectContext(ctx, &posts, query, args...); err != nil {
		return nil, 0, err
	}

	return posts, total, nil
}

func (r *BlogRepository) Update(ctx context.Context, post *domain.BlogPost) error {
	query := `
		UPDATE blog_posts 
		SET slug = $1, title = $2, excerpt = $3, content = $4, 
		    featured_image_url = $5, category_id = $6, tags = $7,
		    meta_title = $8, meta_description = $9, seo_keywords = $10,
		    reading_time_minutes = $11, status = $12, published_at = $13,
		    updated_at = NOW()
		WHERE id = $14
		RETURNING updated_at`

	err := r.db.DB.QueryRowContext(
		ctx, query,
		post.Slug, 
		post.Title, 
		post.Excerpt, 
		post.Content, 
		post.FeaturedImageURL,
		post.CategoryID, 
		pq.Array(post.Tags),
		post.MetaTitle, 
		post.MetaDescription,
		pq.Array(post.SEOKeywords),
		post.ReadingTimeMinutes, 
		post.Status, 
		post.PublishedAt, 
		post.ID,
	).Scan(&post.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("blog post with id %d not found", post.ID)
		}
		return fmt.Errorf("failed to update blog post: %w", err)
	}

	return nil
}

func (r *BlogRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM blog_posts WHERE id = $1`
	_, err := r.db.DB.ExecContext(ctx, query, id)
	return err
}

func (r *BlogRepository) IncrementViews(ctx context.Context, id int64) error {
	query := `UPDATE blog_posts SET views_count = views_count + 1 WHERE id = $1`
	_, err := r.db.DB.ExecContext(ctx, query, id)
	return err
}

func (r *BlogRepository) GetFeatured(ctx context.Context, limit int) ([]*domain.BlogPost, error) {
	var posts []*domain.BlogPost
	query := `
		SELECT * FROM blog_posts 
		WHERE status = 'published' 
		ORDER BY published_at DESC 
		LIMIT $1`

	err := r.db.DB.SelectContext(ctx, &posts, query, limit)
	return posts, err
}

func (r *BlogRepository) GetPopular(ctx context.Context, limit int) ([]*domain.BlogPost, error) {
	var posts []*domain.BlogPost
	query := `
		SELECT * FROM blog_posts 
		WHERE status = 'published' 
		ORDER BY views_count DESC 
		LIMIT $1`

	err := r.db.DB.SelectContext(ctx, &posts, query, limit)
	return posts, err
}

func (r *BlogRepository) Search(ctx context.Context, searchQuery string, limit int) ([]*domain.BlogPost, error) {
	var posts []*domain.BlogPost
	query := `
		SELECT * FROM blog_posts 
		WHERE status = 'published' 
		  AND to_tsvector('english', title || ' ' || excerpt || ' ' || content) @@ plainto_tsquery('english', $1)
		ORDER BY ts_rank(to_tsvector('english', title || ' ' || excerpt || ' ' || content), plainto_tsquery('english', $1)) DESC
		LIMIT $2`

	err := r.db.DB.SelectContext(ctx, &posts, query, searchQuery, limit)
	return posts, err
}

func (r *BlogRepository) GetAnalytics(ctx context.Context) (map[string]interface{}, error) {
	analytics := make(map[string]interface{})

	// Total posts and views
	var totals struct {
		TotalPosts     int `db:"total_posts"`
		PublishedPosts int `db:"published_posts"`
		DraftPosts     int `db:"draft_posts"`
		TotalViews     int `db:"total_views"`
	}
	totalsQuery := `
		SELECT 
			COUNT(*) as total_posts,
			COUNT(*) FILTER (WHERE status = 'published') as published_posts,
			COUNT(*) FILTER (WHERE status = 'draft') as draft_posts,
			COALESCE(SUM(views_count), 0) as total_views
		FROM blog_posts`
	
	if err := r.db.DB.GetContext(ctx, &totals, totalsQuery); err != nil {
		return nil, err
	}
	analytics["total_posts"] = totals.TotalPosts
	analytics["published_posts"] = totals.PublishedPosts
	analytics["draft_posts"] = totals.DraftPosts
	analytics["total_views"] = totals.TotalViews

	// Popular posts
	var popularPosts []struct {
		ID         int64  `db:"id"`
		Title      string `db:"title"`
		ViewsCount int    `db:"views_count"`
		PublishedAt string `db:"published_at"`
	}
	popularQuery := `
		SELECT id, title, views_count, published_at
		FROM blog_posts
		WHERE status = 'published'
		ORDER BY views_count DESC
		LIMIT 10`
	
	if err := r.db.DB.SelectContext(ctx, &popularPosts, popularQuery); err != nil {
		return nil, err
	}
	analytics["popular_posts"] = popularPosts

	// By category
	var byCategory []struct {
		CategoryID   int64  `db:"category_id"`
		CategoryName string `db:"category_name"`
		PostsCount   int    `db:"posts_count"`
		TotalViews   int    `db:"total_views"`
	}
	categoryQuery := `
		SELECT 
			bp.category_id,
			bc.name as category_name,
			COUNT(bp.id) as posts_count,
			COALESCE(SUM(bp.views_count), 0) as total_views
		FROM blog_posts bp
		LEFT JOIN blog_categories bc ON bp.category_id = bc.id
		WHERE bp.status = 'published'
		GROUP BY bp.category_id, bc.name
		ORDER BY posts_count DESC`
	
	if err := r.db.DB.SelectContext(ctx, &byCategory, categoryQuery); err != nil {
		return nil, err
	}
	analytics["by_category"] = byCategory

	return analytics, nil
}