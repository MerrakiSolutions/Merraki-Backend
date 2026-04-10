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
			price_usd_cents, sale_price_usd_cents,
			file_url, file_size_mb, file_format, preview_url,
			status,
			is_featured, is_bestseller, is_new,
			meta_title, meta_description, meta_keywords,
			current_version, published_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20
		) RETURNING id, created_at, updated_at
	`

	return r.db.QueryRowContext(
		ctx, query,
		template.Name, template.Slug, template.Tagline, template.Description, template.CategoryID,
		template.PriceUSDCents, template.SalePriceUSDCents,
		template.FileURL, template.FileSizeMB, template.FileFormat, template.PreviewURL,
		template.Status,
		template.IsFeatured, template.IsBestseller, template.IsNew,
		template.MetaTitle, template.MetaDescription, pq.Array(template.MetaKeywords),
		template.CurrentVersion, template.PublishedAt,
	).Scan(&template.ID, &template.CreatedAt, &template.UpdatedAt)
}

func (r *TemplateRepository) FindByID(ctx context.Context, id int64) (*domain.Template, error) {
	var template domain.Template
	err := r.db.GetContext(ctx, &template, `SELECT * FROM templates WHERE id = $1`, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &template, err
}

func (r *TemplateRepository) FindBySlug(ctx context.Context, slug string) (*domain.Template, error) {
	var template domain.Template
	err := r.db.GetContext(ctx, &template, `SELECT * FROM templates WHERE slug = $1`, slug)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &template, err
}

func (r *TemplateRepository) FindByName(ctx context.Context, name string) (*domain.Template, error) {
	var template domain.Template
	err := r.db.GetContext(ctx, &template, `SELECT * FROM templates WHERE name = $1`, name)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &template, err
}

func (r *TemplateRepository) GetAll(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*domain.Template, int, error) {
	var templates []*domain.Template
	var total int

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
	err := r.db.GetContext(ctx, &total, fmt.Sprintf("SELECT COUNT(*) FROM templates WHERE %s", whereClause), args...)
	if err != nil {
		return nil, 0, err
	}

	// Sort
	orderBy := "created_at DESC"
	if sortBy, ok := filters["sort"].(string); ok {
		switch sortBy {
		case "price_asc":
			orderBy = "price_usd_cents ASC"
		case "price_desc":
			orderBy = "price_usd_cents DESC"
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
		twr := &domain.TemplateWithRelations{Template: *t}

		if t.CategoryID != nil {
			var category domain.Category
			if err := r.db.GetContext(ctx, &category, "SELECT * FROM categories WHERE id = $1", *t.CategoryID); err == nil {
				twr.Category = &category
			}
		}

		twr.Images, _ = r.GetImages(ctx, t.ID)
		twr.Features, _ = r.GetFeatures(ctx, t.ID)
		twr.Tags, _ = r.GetTags(ctx, t.ID)

		result[i] = twr
	}

	return result, total, nil
}

func (r *TemplateRepository) Update(ctx context.Context, template *domain.Template) error {
	query := `
		UPDATE templates SET
			name = $1, slug = $2, tagline = $3, description = $4, category_id = $5,
			price_usd_cents = $6, sale_price_usd_cents = $7,
			file_url = $8, file_size_mb = $9, file_format = $10, preview_url = $11,
			status = $12,
			is_featured = $13, is_bestseller = $14, is_new = $15,
			meta_title = $16, meta_description = $17, meta_keywords = $18,
			current_version = $19, published_at = $20,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $21
	`

	_, err := r.db.ExecContext(
		ctx, query,
		template.Name, template.Slug, template.Tagline, template.Description, template.CategoryID,
		template.PriceUSDCents, template.SalePriceUSDCents,
		template.FileURL, template.FileSizeMB, template.FileFormat, template.PreviewURL,
		template.Status,
		template.IsFeatured, template.IsBestseller, template.IsNew,
		template.MetaTitle, template.MetaDescription, pq.Array(template.MetaKeywords),
		template.CurrentVersion, template.PublishedAt,
		template.ID,
	)
	return err
}

func (r *TemplateRepository) Patch(ctx context.Context, id int64, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	setParts := []string{}
	args := []interface{}{}
	argPos := 1

	for key, value := range updates {
		setParts = append(setParts, fmt.Sprintf("%s = $%d", key, argPos))
		args = append(args, value)
		argPos++
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

func (r *TemplateRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM templates WHERE id = $1", id)
	return err
}

func (r *TemplateRepository) Search(ctx context.Context, query string, limit int) ([]*domain.Template, error) {
	var templates []*domain.Template
	searchQuery := `
		SELECT * FROM templates
		WHERE (name ILIKE $1 OR description ILIKE $1 OR ARRAY_TO_STRING(meta_keywords, ' ') ILIKE $1)
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

	if err := r.db.GetContext(ctx, &total,
		`SELECT COUNT(*) FROM templates WHERE category_id = $1 AND status = 'active'`, categoryID,
	); err != nil {
		return nil, 0, err
	}

	err := r.db.SelectContext(ctx, &templates, `
		SELECT * FROM templates
		WHERE category_id = $1 AND status = 'active'
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, categoryID, limit, offset)
	return templates, total, err
}

func (r *TemplateRepository) GetByTag(ctx context.Context, tag string, limit, offset int) ([]*domain.Template, int, error) {
	var templates []*domain.Template
	var total int

	if err := r.db.GetContext(ctx, &total, `
		SELECT COUNT(DISTINCT t.id)
		FROM templates t
		JOIN template_tags tt ON t.id = tt.template_id
		WHERE tt.tag = $1 AND t.status = 'active'
	`, tag); err != nil {
		return nil, 0, err
	}

	err := r.db.SelectContext(ctx, &templates, `
		SELECT DISTINCT t.*
		FROM templates t
		JOIN template_tags tt ON t.id = tt.template_id
		WHERE tt.tag = $1 AND t.status = 'active'
		ORDER BY t.created_at DESC
		LIMIT $2 OFFSET $3
	`, tag, limit, offset)
	return templates, total, err
}

func (r *TemplateRepository) GetFeatured(ctx context.Context, limit int) ([]*domain.Template, error) {
	var templates []*domain.Template
	err := r.db.SelectContext(ctx, &templates, `
		SELECT * FROM templates
		WHERE is_featured = true AND status = 'active'
		ORDER BY created_at DESC
		LIMIT $1
	`, limit)
	return templates, err
}

func (r *TemplateRepository) GetBestsellers(ctx context.Context, limit int) ([]*domain.Template, error) {
	var templates []*domain.Template
	err := r.db.SelectContext(ctx, &templates, `
		SELECT * FROM templates
		WHERE is_bestseller = true AND status = 'active'
		ORDER BY downloads_count DESC
		LIMIT $1
	`, limit)
	return templates, err
}

func (r *TemplateRepository) GetNew(ctx context.Context, limit int) ([]*domain.Template, error) {
	var templates []*domain.Template
	err := r.db.SelectContext(ctx, &templates, `
		SELECT * FROM templates
		WHERE is_new = true AND status = 'active'
		ORDER BY created_at DESC
		LIMIT $1
	`, limit)
	return templates, err
}

func (r *TemplateRepository) IncrementDownloads(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "UPDATE templates SET downloads_count = downloads_count + 1 WHERE id = $1", id)
	return err
}

func (r *TemplateRepository) IncrementViews(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "UPDATE templates SET views_count = views_count + 1 WHERE id = $1", id)
	return err
}

// ============================================================================
// Images
// ============================================================================

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
	err := r.db.SelectContext(ctx, &images,
		`SELECT * FROM template_images WHERE template_id = $1 ORDER BY display_order, id`, templateID,
	)
	return images, err
}

func (r *TemplateRepository) DeleteImage(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM template_images WHERE id = $1", id)
	return err
}

// ============================================================================
// Features
// ============================================================================

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
	err := r.db.SelectContext(ctx, &features,
		`SELECT * FROM template_features WHERE template_id = $1 ORDER BY display_order, id`, templateID,
	)
	return features, err
}

func (r *TemplateRepository) DeleteFeature(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM template_features WHERE id = $1", id)
	return err
}

// ============================================================================
// Tags
// ============================================================================

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

func (r *TemplateRepository) ReplaceAllTags(ctx context.Context, templateID int64, tags []string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err = tx.ExecContext(ctx, "DELETE FROM template_tags WHERE template_id = $1", templateID); err != nil {
		return err
	}

	for _, tag := range tags {
		if _, err = tx.ExecContext(ctx,
			"INSERT INTO template_tags (template_id, tag) VALUES ($1, $2)", templateID, tag,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// ============================================================================
// Versions
// ============================================================================

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
	err := r.db.SelectContext(ctx, &versions,
		`SELECT * FROM template_versions WHERE template_id = $1 ORDER BY created_at DESC`, templateID,
	)
	return versions, err
}


func (r *TemplateRepository) GetCurrentVersion(ctx context.Context, templateID int64) (*domain.TemplateVersion, error) {
	var version domain.TemplateVersion
	err := r.db.GetContext(ctx, &version,
		`SELECT * FROM template_versions WHERE template_id = $1 AND is_current = true LIMIT 1`, templateID,
	)
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

	if _, err = tx.ExecContext(ctx,
		"UPDATE template_versions SET is_current = false WHERE template_id = $1", templateID,
	); err != nil {
		return err
	}

	if _, err = tx.ExecContext(ctx,
		"UPDATE template_versions SET is_current = true WHERE id = $1", versionID,
	); err != nil {
		return err
	}

	return tx.Commit()
}