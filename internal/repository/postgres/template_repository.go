package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"
	"github.com/merraki/merraki-backend/internal/domain"
)

type TemplateRepository struct {
	db *Database
}

func NewTemplateRepository(db *Database) *TemplateRepository {
	return &TemplateRepository{db: db}
}

func (r *TemplateRepository) Create(ctx context.Context, template *domain.Template) error {
	query := `
		INSERT INTO templates 
		(slug, title, description, detailed_description, price_inr, thumbnail_url, 
		 preview_urls, file_url, file_size_bytes, category_id, tags, status, 
		 is_featured, meta_title, meta_description, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		RETURNING id, created_at, updated_at`

	return r.db.DB.QueryRowContext(
		ctx, query,
		template.Slug, template.Title, template.Description, template.DetailedDescription,
		template.PriceINR, template.ThumbnailURL, template.PreviewURLs, template.FileURL,
		template.FileSizeBytes, template.CategoryID, template.Tags, template.Status,
		template.IsFeatured, template.MetaTitle, template.MetaDescription, template.CreatedBy,
	).Scan(&template.ID, &template.CreatedAt, &template.UpdatedAt)
}

func (r *TemplateRepository) FindByID(ctx context.Context, id int64) (*domain.Template, error) {
	var template domain.Template
	query := `SELECT * FROM templates WHERE id = $1`

	err := r.db.DB.GetContext(ctx, &template, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &template, err
}

func (r *TemplateRepository) FindBySlug(ctx context.Context, slug string) (*domain.Template, error) {
	var template domain.Template
	query := `SELECT * FROM templates WHERE slug = $1`

	err := r.db.DB.GetContext(ctx, &template, query, slug)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &template, err
}

func (r *TemplateRepository) GetAll(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*domain.Template, int, error) {
	var templates []*domain.Template

	query := `SELECT * FROM templates WHERE 1=1`
	countQuery := `SELECT COUNT(*) FROM templates WHERE 1=1`
	args := []interface{}{}
	argCount := 1

	// Status filter
	if status, ok := filters["status"].(string); ok && status != "" {
		query += fmt.Sprintf(" AND status = $%d", argCount)
		countQuery += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, status)
		argCount++
	}

	// Category filter
	if categoryID, ok := filters["category_id"].(int64); ok && categoryID > 0 {
		query += fmt.Sprintf(" AND category_id = $%d", argCount)
		countQuery += fmt.Sprintf(" AND category_id = $%d", argCount)
		args = append(args, categoryID)
		argCount++
	}

	// Featured filter
	if featured, ok := filters["featured"].(bool); ok && featured {
		query += fmt.Sprintf(" AND is_featured = $%d", argCount)
		countQuery += fmt.Sprintf(" AND is_featured = $%d", argCount)
		args = append(args, true)
		argCount++
	}

	// Search filter
	if search, ok := filters["search"].(string); ok && search != "" {
		query += fmt.Sprintf(" AND (title ILIKE $%d OR description ILIKE $%d)", argCount, argCount)
		countQuery += fmt.Sprintf(" AND (title ILIKE $%d OR description ILIKE $%d)", argCount, argCount)
		args = append(args, "%"+search+"%")
		argCount++
	}

	// Price range
	if minPrice, ok := filters["min_price"].(int); ok && minPrice > 0 {
		query += fmt.Sprintf(" AND price_inr >= $%d", argCount)
		countQuery += fmt.Sprintf(" AND price_inr >= $%d", argCount)
		args = append(args, minPrice)
		argCount++
	}

	if maxPrice, ok := filters["max_price"].(int); ok && maxPrice > 0 {
		query += fmt.Sprintf(" AND price_inr <= $%d", argCount)
		countQuery += fmt.Sprintf(" AND price_inr <= $%d", argCount)
		args = append(args, maxPrice)
		argCount++
	}

	// Tags filter - FIX THIS
	if tags, ok := filters["tags"].([]string); ok && len(tags) > 0 {
		query += fmt.Sprintf(" AND tags && $%d", argCount)
		countQuery += fmt.Sprintf(" AND tags && $%d", argCount)
		args = append(args, pq.Array(tags)) // USE pq.Array
		argCount++
	}

	// Sorting
	sort := "created_at DESC"
	if sortParam, ok := filters["sort"].(string); ok {
		switch sortParam {
		case "newest":
			sort = "created_at DESC"
		case "oldest":
			sort = "created_at ASC"
		case "popular":
			sort = "downloads_count DESC"
		case "price_low":
			sort = "price_inr ASC"
		case "price_high":
			sort = "price_inr DESC"
		case "rating":
			sort = "rating DESC"
		}
	}

	query += " ORDER BY " + sort
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argCount, argCount+1)

	// Get total count - ADD ERROR LOGGING
	var total int
	if err := r.db.DB.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("count query failed: %w", err)
	}

	// Get templates - ADD ERROR LOGGING
	args = append(args, limit, offset)
	if err := r.db.DB.SelectContext(ctx, &templates, query, args...); err != nil {
		return nil, 0, fmt.Errorf("select query failed: %w", err)
	}

	return templates, total, nil
}

func (r *TemplateRepository) Update(ctx context.Context, template *domain.Template) error {
	query := `
		UPDATE templates 
		SET slug = $1, title = $2, description = $3, detailed_description = $4, 
		    price_inr = $5, thumbnail_url = $6, preview_urls = $7, file_url = $8,
		    file_size_bytes = $9, category_id = $10, tags = $11, status = $12,
		    is_featured = $13, meta_title = $14, meta_description = $15, updated_at = NOW()
		WHERE id = $16
		RETURNING updated_at`

	return r.db.DB.QueryRowContext(
		ctx, query,
		template.Slug, template.Title, template.Description, template.DetailedDescription,
		template.PriceINR, template.ThumbnailURL, template.PreviewURLs, template.FileURL,
		template.FileSizeBytes, template.CategoryID, template.Tags, template.Status,
		template.IsFeatured, template.MetaTitle, template.MetaDescription, template.ID,
	).Scan(&template.UpdatedAt)
}

func (r *TemplateRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM templates WHERE id = $1`
	
	result, err := r.db.DB.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("template with id %d not found", id)
	}
	
	return nil
}

func (r *TemplateRepository) IncrementViews(ctx context.Context, id int64) error {
	query := `UPDATE templates SET views_count = views_count + 1 WHERE id = $1`
	_, err := r.db.DB.ExecContext(ctx, query, id)
	return err
}

func (r *TemplateRepository) IncrementDownloads(ctx context.Context, id int64) error {
	query := `UPDATE templates SET downloads_count = downloads_count + 1 WHERE id = $1`
	_, err := r.db.DB.ExecContext(ctx, query, id)
	return err
}

func (r *TemplateRepository) GetFeatured(ctx context.Context, limit int) ([]*domain.Template, error) {
	var templates []*domain.Template
	query := `
		SELECT * FROM templates 
		WHERE status = 'active' AND is_featured = true 
		ORDER BY created_at DESC 
		LIMIT $1`

	err := r.db.DB.SelectContext(ctx, &templates, query, limit)
	return templates, err
}

func (r *TemplateRepository) GetPopular(ctx context.Context, limit int) ([]*domain.Template, error) {
	var templates []*domain.Template
	query := `
		SELECT * FROM templates 
		WHERE status = 'active' 
		ORDER BY downloads_count DESC 
		LIMIT $1`

	err := r.db.DB.SelectContext(ctx, &templates, query, limit)
	return templates, err
}

func (r *TemplateRepository) Search(ctx context.Context, searchQuery string, limit int) ([]*domain.Template, error) {
	var templates []*domain.Template
	query := `
		SELECT * FROM templates 
		WHERE status = 'active' 
		  AND to_tsvector('english', title || ' ' || COALESCE(description, '')) @@ plainto_tsquery('english', $1)
		ORDER BY ts_rank(to_tsvector('english', title || ' ' || COALESCE(description, '')), plainto_tsquery('english', $1)) DESC
		LIMIT $2`

	err := r.db.DB.SelectContext(ctx, &templates, query, searchQuery, limit)
	return templates, err
}

func (r *TemplateRepository) GetByIDs(ctx context.Context, ids []int64) ([]*domain.Template, error) {
	if len(ids) == 0 {
		return []*domain.Template{}, nil
	}

	var templates []*domain.Template
	query := `SELECT * FROM templates WHERE id = ANY($1) AND status = 'active'`

	err := r.db.DB.SelectContext(ctx, &templates, query, ids)
	return templates, err
}

func (r *TemplateRepository) GetAnalytics(ctx context.Context) (map[string]interface{}, error) {
	analytics := make(map[string]interface{})

	// 1. Count by status
	type StatusCount struct {
		Status string `db:"status"`
		Count  int    `db:"count"`
	}
	var statusCounts []StatusCount
	statusQuery := `SELECT status, COUNT(*) as count FROM templates GROUP BY status`
	if err := r.db.DB.SelectContext(ctx, &statusCounts, statusQuery); err != nil {
		return nil, fmt.Errorf("failed to get status counts: %w", err)
	}
	analytics["by_status"] = statusCounts

	// 2. Count by category
	type CategoryCount struct {
		CategoryID   int64  `db:"category_id"`
		CategoryName string `db:"category_name"`
		Count        int    `db:"count"`
	}
	var categoryCounts []CategoryCount
	categoryQuery := `
		SELECT 
			t.category_id,
			tc.name as category_name,
			COUNT(t.id) as count
		FROM templates t
		LEFT JOIN template_categories tc ON t.category_id = tc.id
		WHERE t.status = 'active'
		GROUP BY t.category_id, tc.name
		ORDER BY count DESC`
	if err := r.db.DB.SelectContext(ctx, &categoryCounts, categoryQuery); err != nil {
		return nil, fmt.Errorf("failed to get category counts: %w", err)
	}
	analytics["by_category"] = categoryCounts

	// 3. Top templates by downloads
	type TopTemplate struct {
		ID         int64  `db:"id"`
		Title      string `db:"title"`
		Downloads  int    `db:"downloads_count"`
		Views      int    `db:"views_count"`
		Rating     float64 `db:"rating"`
	}
	var topTemplates []TopTemplate
	topQuery := `
		SELECT 
			id, title, downloads_count, views_count, rating
		FROM templates
		WHERE status = 'active'
		ORDER BY downloads_count DESC
		LIMIT 10`
	if err := r.db.DB.SelectContext(ctx, &topTemplates, topQuery); err != nil {
		return nil, fmt.Errorf("failed to get top templates: %w", err)
	}
	analytics["top_templates"] = topTemplates

	// 4. Total counts
	type Totals struct {
		TotalTemplates  int `db:"total_templates"`
		ActiveTemplates int `db:"active_templates"`
		TotalDownloads  int `db:"total_downloads"`
		TotalViews      int `db:"total_views"`
	}
	var totals Totals
	totalsQuery := `
		SELECT 
			COUNT(*) as total_templates,
			COUNT(*) FILTER (WHERE status = 'active') as active_templates,
			COALESCE(SUM(downloads_count), 0) as total_downloads,
			COALESCE(SUM(views_count), 0) as total_views
		FROM templates`
	if err := r.db.DB.GetContext(ctx, &totals, totalsQuery); err != nil {
		return nil, fmt.Errorf("failed to get totals: %w", err)
	}
	analytics["totals"] = totals

	// 5. Revenue by template (simplified - no JOIN to order_items yet)
	type RevenueTemplate struct {
		TemplateID int64  `db:"template_id"`
		Title      string `db:"title"`
		SoldCount  int    `db:"sold_count"`
		Revenue    int64  `db:"revenue"`
	}
	var revenueTemplates []RevenueTemplate
	
	// Check if order_items table has data
	var orderItemsCount int
	if err := r.db.DB.GetContext(ctx, &orderItemsCount, `SELECT COUNT(*) FROM order_items`); err == nil && orderItemsCount > 0 {
		revenueQuery := `
			SELECT 
				oi.template_id,
				t.title,
				COUNT(oi.id) as sold_count,
				SUM(oi.price_inr) as revenue
			FROM order_items oi
			LEFT JOIN templates t ON oi.template_id = t.id
			WHERE oi.template_id IS NOT NULL
			GROUP BY oi.template_id, t.title
			ORDER BY revenue DESC
			LIMIT 10`
		if err := r.db.DB.SelectContext(ctx, &revenueTemplates, revenueQuery); err != nil {
			// If error, just set empty array
			revenueTemplates = []RevenueTemplate{}
		}
	}
	analytics["revenue_by_template"] = revenueTemplates

	// 6. Recent templates
	type RecentTemplate struct {
		ID        int64     `db:"id"`
		Title     string    `db:"title"`
		Status    string    `db:"status"`
		CreatedAt time.Time `db:"created_at"`
	}
	var recentTemplates []RecentTemplate
	recentQuery := `
		SELECT id, title, status, created_at
		FROM templates
		ORDER BY created_at DESC
		LIMIT 5`
	if err := r.db.DB.SelectContext(ctx, &recentTemplates, recentQuery); err != nil {
		return nil, fmt.Errorf("failed to get recent templates: %w", err)
	}
	analytics["recent_templates"] = recentTemplates

	return analytics, nil
}