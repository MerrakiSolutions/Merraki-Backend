package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/lib/pq"
	"github.com/merraki/merraki-backend/internal/domain"
)

type BlogPostRepository struct {
	db *Database
}

func NewBlogPostRepository(db *Database) *BlogPostRepository {
	return &BlogPostRepository{db: db}
}

func (r *BlogPostRepository) Create(ctx context.Context, post *domain.BlogPost) error {
	query := `
		INSERT INTO blog_posts 
		(title, slug, excerpt, content, featured_image_url, author_id, category_id,
		 tags, meta_title, meta_description, meta_keywords, status, is_featured,
		 reading_time_minutes, published_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id, views_count, created_at, updated_at`

	err := r.db.DB.QueryRowContext(
		ctx, query,
		post.Title, post.Slug, post.Excerpt, post.Content, post.FeaturedImageURL,
		post.AuthorID, post.CategoryID, pq.Array(post.Tags), post.MetaTitle,
		post.MetaDescription, pq.Array(post.MetaKeywords), post.Status,
		post.IsFeatured, post.ReadingTimeMinutes, post.PublishedAt,
	).Scan(&post.ID, &post.ViewsCount, &post.CreatedAt, &post.UpdatedAt)

	return err
}

func (r *BlogPostRepository) FindByID(ctx context.Context, id int64) (*domain.BlogPost, error) {
	var post domain.BlogPost
	query := `SELECT * FROM blog_posts WHERE id = $1`

	err := r.db.DB.GetContext(ctx, &post, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &post, err
}

func (r *BlogPostRepository) FindBySlug(ctx context.Context, slug string) (*domain.BlogPost, error) {
	var post domain.BlogPost
	query := `SELECT * FROM blog_posts WHERE slug = $1`

	err := r.db.DB.GetContext(ctx, &post, query, slug)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &post, err
}

// GetAll with comprehensive filters
func (r *BlogPostRepository) GetAll(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*domain.BlogPost, int, error) {
	var posts []*domain.BlogPost
	
	query := `SELECT * FROM blog_posts WHERE 1=1`
	countQuery := `SELECT COUNT(*) FROM blog_posts WHERE 1=1`
	args := []interface{}{}
	argCount := 1

	// Status filter
	if status, ok := filters["status"].(string); ok && status != "" {
		query += fmt.Sprintf(" AND status = $%d", argCount)
		countQuery += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, status)
		argCount++
	}

	// Author filter
	if authorID, ok := filters["author_id"].(int64); ok && authorID > 0 {
		query += fmt.Sprintf(" AND author_id = $%d", argCount)
		countQuery += fmt.Sprintf(" AND author_id = $%d", argCount)
		args = append(args, authorID)
		argCount++
	}

	// Category filter
	if categoryID, ok := filters["category_id"].(int64); ok && categoryID > 0 {
		query += fmt.Sprintf(" AND category_id = $%d", argCount)
		countQuery += fmt.Sprintf(" AND category_id = $%d", argCount)
		args = append(args, categoryID)
		argCount++
	}

	// Tag filter
	if tag, ok := filters["tag"].(string); ok && tag != "" {
		query += fmt.Sprintf(" AND $%d = ANY(tags)", argCount)
		countQuery += fmt.Sprintf(" AND $%d = ANY(tags)", argCount)
		args = append(args, tag)
		argCount++
	}

	// Search filter (title, excerpt, content)
	if search, ok := filters["search"].(string); ok && search != "" {
		query += fmt.Sprintf(" AND (title ILIKE $%d OR excerpt ILIKE $%d OR content ILIKE $%d)", argCount, argCount, argCount)
		countQuery += fmt.Sprintf(" AND (title ILIKE $%d OR excerpt ILIKE $%d OR content ILIKE $%d)", argCount, argCount, argCount)
		args = append(args, "%"+search+"%")
		argCount++
	}

	// Featured filter
	if featured, ok := filters["featured"].(bool); ok {
		query += fmt.Sprintf(" AND is_featured = $%d", argCount)
		countQuery += fmt.Sprintf(" AND is_featured = $%d", argCount)
		args = append(args, featured)
		argCount++
	}

	// Date range filter
	if startDate, ok := filters["start_date"].(string); ok && startDate != "" {
		query += fmt.Sprintf(" AND created_at >= $%d", argCount)
		countQuery += fmt.Sprintf(" AND created_at >= $%d", argCount)
		args = append(args, startDate)
		argCount++
	}

	if endDate, ok := filters["end_date"].(string); ok && endDate != "" {
		query += fmt.Sprintf(" AND created_at <= $%d", argCount)
		countQuery += fmt.Sprintf(" AND created_at <= $%d", argCount)
		args = append(args, endDate)
		argCount++
	}

	// Sorting
	sortOrder := "created_at DESC"
	if sort, ok := filters["sort"].(string); ok {
		switch sort {
		case "title_asc":
			sortOrder = "title ASC"
		case "title_desc":
			sortOrder = "title DESC"
		case "published_asc":
			sortOrder = "published_at ASC"
		case "published_desc":
			sortOrder = "published_at DESC"
		case "views_asc":
			sortOrder = "views_count ASC"
		case "views_desc":
			sortOrder = "views_count DESC"
		case "oldest":
			sortOrder = "created_at ASC"
		case "newest":
			sortOrder = "created_at DESC"
		}
	}

	query += " ORDER BY " + sortOrder
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argCount, argCount+1)

	// Get total count
	var total int
	if err := r.db.DB.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("count query failed: %w", err)
	}

	// Get posts
	args = append(args, limit, offset)
	if err := r.db.DB.SelectContext(ctx, &posts, query, args...); err != nil {
		return nil, 0, fmt.Errorf("select query failed: %w", err)
	}

	return posts, total, nil
}

// GetAllWithRelations - Get posts with author and category info
func (r *BlogPostRepository) GetAllWithRelations(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*domain.BlogPostWithRelations, int, error) {
	query := `
		SELECT 
			p.*,
			a.id as "author.id", a.name as "author.name", a.slug as "author.slug",
			a.email as "author.email", a.bio as "author.bio", a.avatar_url as "author.avatar_url",
			c.id as "category.id", c.name as "category.name", c.slug as "category.slug",
			c.description as "category.description"
		FROM blog_posts p
		LEFT JOIN blog_authors a ON p.author_id = a.id
		LEFT JOIN blog_categories c ON p.category_id = c.id
		WHERE 1=1`

	countQuery := `SELECT COUNT(*) FROM blog_posts p WHERE 1=1`
	args := []interface{}{}
	argCount := 1

	// Apply same filters as GetAll
	if status, ok := filters["status"].(string); ok && status != "" {
		query += fmt.Sprintf(" AND p.status = $%d", argCount)
		countQuery += fmt.Sprintf(" AND p.status = $%d", argCount)
		args = append(args, status)
		argCount++
	}

	if authorID, ok := filters["author_id"].(int64); ok && authorID > 0 {
		query += fmt.Sprintf(" AND p.author_id = $%d", argCount)
		countQuery += fmt.Sprintf(" AND p.author_id = $%d", argCount)
		args = append(args, authorID)
		argCount++
	}

	if categoryID, ok := filters["category_id"].(int64); ok && categoryID > 0 {
		query += fmt.Sprintf(" AND p.category_id = $%d", argCount)
		countQuery += fmt.Sprintf(" AND p.category_id = $%d", argCount)
		args = append(args, categoryID)
		argCount++
	}

	if search, ok := filters["search"].(string); ok && search != "" {
		query += fmt.Sprintf(" AND (p.title ILIKE $%d OR p.excerpt ILIKE $%d)", argCount, argCount)
		countQuery += fmt.Sprintf(" AND (p.title ILIKE $%d OR p.excerpt ILIKE $%d)", argCount, argCount)
		args = append(args, "%"+search+"%")
		argCount++
	}

	sortOrder := "p.created_at DESC"
	if sort, ok := filters["sort"].(string); ok && sort != "" {
		switch sort {
		case "newest":
			sortOrder = "p.created_at DESC"
		case "oldest":
			sortOrder = "p.created_at ASC"
		case "title_asc":
			sortOrder = "p.title ASC"
		case "title_desc":
			sortOrder = "p.title DESC"
		}
	}

	query += " ORDER BY " + sortOrder
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argCount, argCount+1)

	// Get total count
	var total int
	if err := r.db.DB.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	// Get posts with relations
	args = append(args, limit, offset)
	rows, err := r.db.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var posts []*domain.BlogPostWithRelations
	for rows.Next() {
		var post domain.BlogPostWithRelations
		var author domain.BlogAuthor
		var category domain.BlogCategory

		err := rows.Scan(
			// Post fields
			&post.ID, &post.Title, &post.Slug, &post.Excerpt, &post.Content,
			&post.FeaturedImageURL, &post.AuthorID, &post.CategoryID,
			pq.Array(&post.Tags), &post.MetaTitle, &post.MetaDescription,
			pq.Array(&post.MetaKeywords), &post.Status, &post.IsFeatured,
			&post.ViewsCount, &post.ReadingTimeMinutes, &post.PublishedAt,
			&post.CreatedAt, &post.UpdatedAt,
			// Author fields
			&author.ID, &author.Name, &author.Slug, &author.Email,
			&author.Bio, &author.AvatarURL,
			// Category fields
			&category.ID, &category.Name, &category.Slug, &category.Description,
		)
		if err != nil {
			return nil, 0, err
		}

		if author.ID > 0 {
			post.Author = &author
		}
		if category.ID > 0 {
			post.Category = &category
		}

		posts = append(posts, &post)
	}

	return posts, total, nil
}

func (r *BlogPostRepository) Update(ctx context.Context, post *domain.BlogPost) error {
	query := `
		UPDATE blog_posts 
		SET title = $1, slug = $2, excerpt = $3, content = $4,
		    featured_image_url = $5, author_id = $6, category_id = $7,
		    tags = $8, meta_title = $9, meta_description = $10,
		    meta_keywords = $11, status = $12, is_featured = $13,
		    reading_time_minutes = $14, published_at = $15, updated_at = NOW()
		WHERE id = $16
		RETURNING updated_at`

	err := r.db.DB.QueryRowContext(
		ctx, query,
		post.Title, post.Slug, post.Excerpt, post.Content, post.FeaturedImageURL,
		post.AuthorID, post.CategoryID, pq.Array(post.Tags), post.MetaTitle,
		post.MetaDescription, pq.Array(post.MetaKeywords), post.Status,
		post.IsFeatured, post.ReadingTimeMinutes, post.PublishedAt, post.ID,
	).Scan(&post.UpdatedAt)

	return err
}

// Patch - Partial update
func (r *BlogPostRepository) Patch(ctx context.Context, id int64, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	setParts := []string{}
	args := []interface{}{}
	argCount := 1

	for key, value := range updates {
		setParts = append(setParts, fmt.Sprintf("%s = $%d", key, argCount))
		args = append(args, value)
		argCount++
	}

	setParts = append(setParts, "updated_at = NOW()")
	args = append(args, id)

	query := fmt.Sprintf("UPDATE blog_posts SET %s WHERE id = $%d", strings.Join(setParts, ", "), argCount)

	_, err := r.db.DB.ExecContext(ctx, query, args...)
	return err
}

func (r *BlogPostRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM blog_posts WHERE id = $1`
	_, err := r.db.DB.ExecContext(ctx, query, id)
	return err
}

func (r *BlogPostRepository) IncrementViews(ctx context.Context, id int64) error {
	query := `UPDATE blog_posts SET views_count = views_count + 1 WHERE id = $1`
	_, err := r.db.DB.ExecContext(ctx, query, id)
	return err
}

func (r *BlogPostRepository) Search(ctx context.Context, searchTerm string, limit int) ([]*domain.BlogPost, error) {
	var posts []*domain.BlogPost
	query := `
		SELECT * FROM blog_posts 
		WHERE status = 'published'
		  AND to_tsvector('english', title || ' ' || COALESCE(excerpt, '') || ' ' || content) @@ plainto_tsquery('english', $1)
		ORDER BY ts_rank(to_tsvector('english', title || ' ' || content), plainto_tsquery('english', $1)) DESC
		LIMIT $2`

	err := r.db.DB.SelectContext(ctx, &posts, query, searchTerm, limit)
	return posts, err
}

func (r *BlogPostRepository) GetByAuthor(ctx context.Context, authorID int64, limit, offset int) ([]*domain.BlogPost, int, error) {
	var posts []*domain.BlogPost
	
	query := `SELECT * FROM blog_posts WHERE author_id = $1 AND status = 'published' ORDER BY published_at DESC LIMIT $2 OFFSET $3`
	countQuery := `SELECT COUNT(*) FROM blog_posts WHERE author_id = $1 AND status = 'published'`

	var total int
	if err := r.db.DB.GetContext(ctx, &total, countQuery, authorID); err != nil {
		return nil, 0, err
	}

	if err := r.db.DB.SelectContext(ctx, &posts, query, authorID, limit, offset); err != nil {
		return nil, 0, err
	}

	return posts, total, nil
}

func (r *BlogPostRepository) GetByCategory(ctx context.Context, categoryID int64, limit, offset int) ([]*domain.BlogPost, int, error) {
	var posts []*domain.BlogPost
	
	query := `SELECT * FROM blog_posts WHERE category_id = $1 AND status = 'published' ORDER BY published_at DESC LIMIT $2 OFFSET $3`
	countQuery := `SELECT COUNT(*) FROM blog_posts WHERE category_id = $1 AND status = 'published'`

	var total int
	if err := r.db.DB.GetContext(ctx, &total, countQuery, categoryID); err != nil {
		return nil, 0, err
	}

	if err := r.db.DB.SelectContext(ctx, &posts, query, categoryID, limit, offset); err != nil {
		return nil, 0, err
	}

	return posts, total, nil
}

func (r *BlogPostRepository) GetByTag(ctx context.Context, tag string, limit, offset int) ([]*domain.BlogPost, int, error) {
	var posts []*domain.BlogPost
	
	query := `SELECT * FROM blog_posts WHERE $1 = ANY(tags) AND status = 'published' ORDER BY published_at DESC LIMIT $2 OFFSET $3`
	countQuery := `SELECT COUNT(*) FROM blog_posts WHERE $1 = ANY(tags) AND status = 'published'`

	var total int
	if err := r.db.DB.GetContext(ctx, &total, countQuery, tag); err != nil {
		return nil, 0, err
	}

	if err := r.db.DB.SelectContext(ctx, &posts, query, tag, limit, offset); err != nil {
		return nil, 0, err
	}

	return posts, total, nil
}