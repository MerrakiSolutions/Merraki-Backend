package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/merraki/merraki-backend/internal/domain"
)

type TemplateRepository struct {
	db *sqlx.DB
}

func NewTemplateRepository(db *sqlx.DB) *TemplateRepository {
	return &TemplateRepository{db: db}
}

func (r *TemplateRepository) Create(ctx context.Context, template *domain.Template) error {
	query := `
		INSERT INTO templates (
			name, slug, tagline, description, category_id,
			price, sale_price, is_on_sale,
			file_url, file_size_mb, file_format, preview_url,
			stock_quantity, is_unlimited_stock, status, is_available,
			is_featured, is_bestseller, is_new,
			meta_title, meta_description, meta_keywords,
			current_version, published_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18,
			$19, $20, $21, $22, $23, $24
		) RETURNING id, created_at, updated_at
	`

	return r.db.QueryRowContext(
		ctx, query,
		template.Name, template.Slug, template.Tagline, template.Description, template.CategoryID,
		template.Price, template.SalePrice, template.IsOnSale,
		template.FileURL, template.FileSizeMB, template.FileFormat, template.PreviewURL,
		template.StockQuantity, template.IsUnlimitedStock, template.Status, template.IsAvailable,
		template.IsFeatured, template.IsBestseller, template.IsNew,
		template.MetaTitle, template.MetaDescription, pq.Array(template.MetaKeywords),
		template.CurrentVersion, template.PublishedAt,
	).Scan(&template.ID, &template.CreatedAt, &template.UpdatedAt)
}

func (r *TemplateRepository) FindByID(ctx context.Context, id int64) (*domain.Template, error) {
	var template domain.Template
	query := `SELECT * FROM templates WHERE id = $1`

	err := r.db.GetContext(ctx, &template, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &template, err
}

func (r *TemplateRepository) FindBySlug(ctx context.Context, slug string) (*domain.Template, error) {
	var template domain.Template
	query := `SELECT * FROM templates WHERE slug = $1`

	err := r.db.GetContext(ctx, &template, query, slug)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &template, err
}

func (r *TemplateRepository) GetAll(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*domain.Template, int, error) {
	var templates []*domain.Template
	var total int

	// Build WHERE clause
	whereClauses := []string{"1=1"}
	args := []interface{}{}
	argPos := 1

	if categoryID, ok := filters["category_id"].(int64); ok {
		whereClauses = append(whereClauses, fmt.Sprintf("category_id = $%d", argPos))
		args = append(args, categoryID)
		argPos++
	}

	if status, ok := filters["status"].(domain.TemplateStatus); ok {
		whereClauses = append(whereClauses, fmt.Sprintf("status = $%d", argPos))
		args = append(args, status)
		argPos++
	}

	if available, ok := filters["available"].(bool); ok {
		whereClauses = append(whereClauses, fmt.Sprintf("is_available = $%d", argPos))
		args = append(args, available)
		argPos++
	}

	if featured, ok := filters["featured"].(bool); ok && featured {
		whereClauses = append(whereClauses, fmt.Sprintf("is_featured = $%d", argPos))
		args = append(args, true)
		argPos++
	}

	if search, ok := filters["search"].(string); ok && search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(name ILIKE $%d OR description ILIKE $%d)", argPos, argPos))
		args = append(args, "%"+search+"%")
		argPos++
	}

	whereClause := strings.Join(whereClauses, " AND ")

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM templates WHERE %s", whereClause)
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	// Get data
	orderBy := "created_at DESC"
	if sortBy, ok := filters["sort"].(string); ok {
		switch sortBy {
		case "price_asc":
			orderBy = "price ASC"
		case "price_desc":
			orderBy = "price DESC"
		case "popular":
			orderBy = "downloads_count DESC"
		case "newest":
			orderBy = "created_at DESC"
		}
	}

	args = append(args, limit, offset)
	query := fmt.Sprintf(`
		SELECT * FROM templates 
		WHERE %s 
		ORDER BY %s 
		LIMIT $%d OFFSET $%d
	`, whereClause, orderBy, argPos, argPos+1)

	err = r.db.SelectContext(ctx, &templates, query, args...)
	return templates, total, err
}

func (r *TemplateRepository) GetAllWithRelations(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*domain.TemplateWithRelations, int, error) {
	templates, total, err := r.GetAll(ctx, filters, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	result := make([]*domain.TemplateWithRelations, len(templates))
	for i, t := range templates {
		twr := &domain.TemplateWithRelations{
			Template: *t,
		}

		// Load category
		if t.CategoryID != nil {
			var category domain.Category
			err := r.db.GetContext(ctx, &category, "SELECT * FROM categories WHERE id = $1", *t.CategoryID)
			if err == nil {
				twr.Category = &category
			}
		}

		// Load images
		images, _ := r.GetImages(ctx, t.ID)
		twr.Images = images

		// Load features
		features, _ := r.GetFeatures(ctx, t.ID)
		twr.Features = features

		// Load tags
		tags, _ := r.GetTags(ctx, t.ID)
		twr.Tags = tags

		result[i] = twr
	}

	return result, total, nil
}

// Add these methods to the existing TemplateRepository

func (r *TemplateRepository) FindByName(ctx context.Context, name string) (*domain.Template, error) {
	var template domain.Template
	query := `SELECT * FROM templates WHERE name = $1`

	err := r.db.GetContext(ctx, &template, query, name)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &template, err
}

func (r *TemplateRepository) Patch(ctx context.Context, id int64, updates map[string]interface{}) error {
	// Build dynamic UPDATE query
	setParts := []string{}
	args := []interface{}{}
	argPos := 1

	for key, value := range updates {
		setParts = append(setParts, fmt.Sprintf("%s = $%d", key, argPos))
		args = append(args, value)
		argPos++
	}

	if len(setParts) == 0 {
		return nil
	}

	args = append(args, id)
	query := fmt.Sprintf(
		"UPDATE templates SET %s, updated_at = CURRENT_TIMESTAMP WHERE id = $%d",
		strings.Join(setParts, ", "),
		argPos,
	)

	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *TemplateRepository) Search(ctx context.Context, query string, limit int) ([]*domain.Template, error) {
	var templates []*domain.Template
	searchQuery := `
		SELECT * FROM templates 
		WHERE (name ILIKE $1 OR description ILIKE $1 OR ARRAY_TO_STRING(meta_keywords, ' ') ILIKE $1)
		AND is_available = true
		AND status = 'active'
		ORDER BY 
			CASE 
				WHEN name ILIKE $1 THEN 1
				WHEN description ILIKE $1 THEN 2
				ELSE 3
			END,
			downloads_count DESC
		LIMIT $2
	`
	err := r.db.SelectContext(ctx, &templates, searchQuery, "%"+query+"%", limit)
	return templates, err
}

func (r *TemplateRepository) GetByCategory(ctx context.Context, categoryID int64, limit, offset int) ([]*domain.Template, int, error) {
	var templates []*domain.Template
	var total int

	// Count
	countQuery := `SELECT COUNT(*) FROM templates WHERE category_id = $1 AND is_available = true`
	if err := r.db.GetContext(ctx, &total, countQuery, categoryID); err != nil {
		return nil, 0, err
	}

	// Get data
	query := `
		SELECT * FROM templates 
		WHERE category_id = $1 AND is_available = true
		ORDER BY created_at DESC 
		LIMIT $2 OFFSET $3
	`
	err := r.db.SelectContext(ctx, &templates, query, categoryID, limit, offset)
	return templates, total, err
}

func (r *TemplateRepository) GetByTag(ctx context.Context, tag string, limit, offset int) ([]*domain.Template, int, error) {
	var templates []*domain.Template
	var total int

	// Count
	countQuery := `
		SELECT COUNT(DISTINCT t.*) 
		FROM templates t
		JOIN template_tags tt ON t.id = tt.template_id
		WHERE tt.tag = $1 AND t.is_available = true
	`
	if err := r.db.GetContext(ctx, &total, countQuery, tag); err != nil {
		return nil, 0, err
	}

	// Get data
	query := `
		SELECT DISTINCT t.* 
		FROM templates t
		JOIN template_tags tt ON t.id = tt.template_id
		WHERE tt.tag = $1 AND t.is_available = true
		ORDER BY t.created_at DESC 
		LIMIT $2 OFFSET $3
	`
	err := r.db.SelectContext(ctx, &templates, query, tag, limit, offset)
	return templates, total, err
}

func (r *TemplateRepository) GetFeatured(ctx context.Context, limit int) ([]*domain.Template, error) {
	var templates []*domain.Template
	query := `
		SELECT * FROM templates 
		WHERE is_featured = true AND is_available = true AND status = 'active'
		ORDER BY created_at DESC 
		LIMIT $1
	`
	err := r.db.SelectContext(ctx, &templates, query, limit)
	return templates, err
}

func (r *TemplateRepository) GetBestsellers(ctx context.Context, limit int) ([]*domain.Template, error) {
	var templates []*domain.Template
	query := `
		SELECT * FROM templates 
		WHERE is_bestseller = true AND is_available = true AND status = 'active'
		ORDER BY downloads_count DESC 
		LIMIT $1
	`
	err := r.db.SelectContext(ctx, &templates, query, limit)
	return templates, err
}

func (r *TemplateRepository) GetNew(ctx context.Context, limit int) ([]*domain.Template, error) {
	var templates []*domain.Template
	query := `
		SELECT * FROM templates 
		WHERE is_new = true AND is_available = true AND status = 'active'
		ORDER BY created_at DESC 
		LIMIT $1
	`
	err := r.db.SelectContext(ctx, &templates, query, limit)
	return templates, err
}

func (r *TemplateRepository) ReplaceAllTags(ctx context.Context, templateID int64, tags []string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete existing tags
	_, err = tx.ExecContext(ctx, "DELETE FROM template_tags WHERE template_id = $1", templateID)
	if err != nil {
		return err
	}

	// Insert new tags
	for _, tag := range tags {
		_, err = tx.ExecContext(ctx,
			"INSERT INTO template_tags (template_id, tag) VALUES ($1, $2)",
			templateID, tag,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *TemplateRepository) CreateAnalyticsEvent(ctx context.Context, event *domain.TemplateAnalytics) error {
	// This method would insert into a template_analytics table if you have one
	// For now, we'll just return nil
	return nil
}

func (r *TemplateRepository) Update(ctx context.Context, template *domain.Template) error {
	query := `
		UPDATE templates SET
			name = $1, slug = $2, tagline = $3, description = $4, category_id = $5,
			price = $6, sale_price = $7, is_on_sale = $8,
			file_url = $9, file_size_mb = $10, file_format = $11, preview_url = $12,
			stock_quantity = $13, is_unlimited_stock = $14, status = $15, is_available = $16,
			is_featured = $17, is_bestseller = $18, is_new = $19,
			meta_title = $20, meta_description = $21, meta_keywords = $22,
			current_version = $23, published_at = $24
		WHERE id = $25
	`

	_, err := r.db.ExecContext(
		ctx, query,
		template.Name, template.Slug, template.Tagline, template.Description, template.CategoryID,
		template.Price, template.SalePrice, template.IsOnSale,
		template.FileURL, template.FileSizeMB, template.FileFormat, template.PreviewURL,
		template.StockQuantity, template.IsUnlimitedStock, template.Status, template.IsAvailable,
		template.IsFeatured, template.IsBestseller, template.IsNew,
		template.MetaTitle, template.MetaDescription, pq.Array(template.MetaKeywords),
		template.CurrentVersion, template.PublishedAt,
		template.ID,
	)
	return err
}

func (r *TemplateRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM templates WHERE id = $1", id)
	return err
}

func (r *TemplateRepository) DecrementStock(ctx context.Context, id int64, quantity int) error {
	query := `
		UPDATE templates 
		SET stock_quantity = stock_quantity - $1 
		WHERE id = $2 AND (is_unlimited_stock = true OR stock_quantity >= $1)
	`
	result, err := r.db.ExecContext(ctx, query, quantity, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return domain.ErrInsufficientStock
	}

	return nil
}

func (r *TemplateRepository) IncrementDownloads(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "UPDATE templates SET downloads_count = downloads_count + 1 WHERE id = $1", id)
	return err
}

func (r *TemplateRepository) IncrementViews(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "UPDATE templates SET views_count = views_count + 1 WHERE id = $1", id)
	return err
}

// Images
func (r *TemplateRepository) CreateImage(ctx context.Context, image *domain.TemplateImage) error {
	query := `
		INSERT INTO template_images (template_id, url, alt_text, display_order, is_primary)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`
	return r.db.QueryRowContext(ctx, query,
		image.TemplateID, image.URL, image.AltText, image.DisplayOrder, image.IsPrimary,
	).Scan(&image.ID, &image.CreatedAt)
}

func (r *TemplateRepository) GetImages(ctx context.Context, templateID int64) ([]*domain.TemplateImage, error) {
	var images []*domain.TemplateImage
	query := `SELECT * FROM template_images WHERE template_id = $1 ORDER BY display_order, id`
	err := r.db.SelectContext(ctx, &images, query, templateID)
	return images, err
}

func (r *TemplateRepository) DeleteImage(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM template_images WHERE id = $1", id)
	return err
}

// Features
func (r *TemplateRepository) CreateFeature(ctx context.Context, feature *domain.TemplateFeature) error {
	query := `
		INSERT INTO template_features (template_id, title, description, display_order)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`
	return r.db.QueryRowContext(ctx, query,
		feature.TemplateID, feature.Title, feature.Description, feature.DisplayOrder,
	).Scan(&feature.ID, &feature.CreatedAt)
}

func (r *TemplateRepository) GetFeatures(ctx context.Context, templateID int64) ([]*domain.TemplateFeature, error) {
	var features []*domain.TemplateFeature
	query := `SELECT * FROM template_features WHERE template_id = $1 ORDER BY display_order, id`
	err := r.db.SelectContext(ctx, &features, query, templateID)
	return features, err
}

func (r *TemplateRepository) DeleteFeature(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM template_features WHERE id = $1", id)
	return err
}

// Tags
func (r *TemplateRepository) AddTag(ctx context.Context, templateID int64, tag string) error {
	_, err := r.db.ExecContext(ctx,
		"INSERT INTO template_tags (template_id, tag) VALUES ($1, $2) ON CONFLICT DO NOTHING",
		templateID, tag,
	)
	return err
}

func (r *TemplateRepository) GetTags(ctx context.Context, templateID int64) ([]string, error) {
	var tags []string
	err := r.db.SelectContext(ctx, &tags, "SELECT tag FROM template_tags WHERE template_id = $1", templateID)
	return tags, err
}

func (r *TemplateRepository) RemoveTags(ctx context.Context, templateID int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM template_tags WHERE template_id = $1", templateID)
	return err
}

// Versions
func (r *TemplateRepository) CreateVersion(ctx context.Context, version *domain.TemplateVersion) error {
	query := `
		INSERT INTO template_versions (template_id, version_number, file_url, file_size_mb, changelog, is_current, uploaded_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`
	return r.db.QueryRowContext(ctx, query,
		version.TemplateID, version.VersionNumber, version.FileURL, version.FileSizeMB,
		version.Changelog, version.IsCurrent, version.UploadedBy,
	).Scan(&version.ID, &version.CreatedAt)
}

func (r *TemplateRepository) GetVersions(ctx context.Context, templateID int64) ([]*domain.TemplateVersion, error) {
	var versions []*domain.TemplateVersion
	query := `SELECT * FROM template_versions WHERE template_id = $1 ORDER BY created_at DESC`
	err := r.db.SelectContext(ctx, &versions, query, templateID)
	return versions, err
}

func (r *TemplateRepository) GetCurrentVersion(ctx context.Context, templateID int64) (*domain.TemplateVersion, error) {
	var version domain.TemplateVersion
	query := `SELECT * FROM template_versions WHERE template_id = $1 AND is_current = true LIMIT 1`
	err := r.db.GetContext(ctx, &version, query, templateID)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return &version, err
}

func (r *TemplateRepository) SetCurrentVersion(ctx context.Context, templateID int64, versionID int64) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Unset all current versions
	_, err = tx.ExecContext(ctx, "UPDATE template_versions SET is_current = false WHERE template_id = $1", templateID)
	if err != nil {
		return err
	}

	// Set new current version
	_, err = tx.ExecContext(ctx, "UPDATE template_versions SET is_current = true WHERE id = $1", versionID)
	if err != nil {
		return err
	}

	return tx.Commit()
}
