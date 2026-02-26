package postgres

import (
	"context"
	"database/sql"

	"github.com/merraki/merraki-backend/internal/domain"
)

type CategoryRepository struct {
	db *Database
}

func NewCategoryRepository(db *Database) *CategoryRepository {
	return &CategoryRepository{db: db}
}

// Template Categories
func (r *CategoryRepository) CreateTemplateCategory(ctx context.Context, category *domain.TemplateCategory) error {
	query := `
		INSERT INTO template_categories 
		(slug, name, description, icon_name, display_order, color_hex, meta_title, meta_description, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at`

	return r.db.DB.QueryRowContext(
		ctx, query,
		category.Slug, category.Name, category.Description, category.IconName,
		category.DisplayOrder, category.ColorHex, category.MetaTitle, category.MetaDescription, category.IsActive,
	).Scan(&category.ID, &category.CreatedAt, &category.UpdatedAt)
}

func (r *CategoryRepository) GetTemplateCategories(ctx context.Context, activeOnly bool) ([]*domain.TemplateCategory, error) {
	var categories []*domain.TemplateCategory
	query := `SELECT * FROM template_categories WHERE 1=1`

	if activeOnly {
		query += ` AND is_active = true`
	}

	query += ` ORDER BY display_order ASC, name ASC`

	err := r.db.DB.SelectContext(ctx, &categories, query)
	return categories, err
}

func (r *CategoryRepository) GetTemplateCategoryByID(ctx context.Context, id int64) (*domain.TemplateCategory, error) {
	var category domain.TemplateCategory
	query := `SELECT * FROM template_categories WHERE id = $1`

	err := r.db.DB.GetContext(ctx, &category, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &category, err
}

func (r *CategoryRepository) GetTemplateCategoryBySlug(ctx context.Context, slug string) (*domain.TemplateCategory, error) {
	var category domain.TemplateCategory
	query := `SELECT * FROM template_categories WHERE slug = $1`

	err := r.db.DB.GetContext(ctx, &category, query, slug)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &category, err
}

func (r *CategoryRepository) UpdateTemplateCategory(ctx context.Context, category *domain.TemplateCategory) error {
	query := `
		UPDATE template_categories 
		SET name = $1, description = $2, icon_name = $3, display_order = $4, 
		    color_hex = $5, meta_title = $6, meta_description = $7, is_active = $8, updated_at = NOW()
		WHERE id = $9
		RETURNING updated_at`

	return r.db.DB.QueryRowContext(
		ctx, query,
		category.Name, category.Description, category.IconName, category.DisplayOrder,
		category.ColorHex, category.MetaTitle, category.MetaDescription, category.IsActive, category.ID,
	).Scan(&category.UpdatedAt)
}

func (r *CategoryRepository) DeleteTemplateCategory(ctx context.Context, id int64) error {
	query := `DELETE FROM template_categories WHERE id = $1`
	_, err := r.db.DB.ExecContext(ctx, query, id)
	return err
}

// Blog Categories
func (r *CategoryRepository) CreateBlogCategory(ctx context.Context, category *domain.BlogCategory) error {
	query := `
		INSERT INTO blog_categories 
		(slug, name, description, display_order, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`

	return r.db.DB.QueryRowContext(
		ctx, query,
		category.Slug, category.Name, category.Description, category.DisplayOrder, category.IsActive,
	).Scan(&category.ID, &category.CreatedAt, &category.UpdatedAt)
}

func (r *CategoryRepository) GetBlogCategories(ctx context.Context, activeOnly bool) ([]*domain.BlogCategory, error) {
	var categories []*domain.BlogCategory
	query := `SELECT * FROM blog_categories WHERE 1=1`

	if activeOnly {
		query += ` AND is_active = true`
	}

	query += ` ORDER BY display_order ASC, name ASC`

	err := r.db.DB.SelectContext(ctx, &categories, query)
	return categories, err
}

func (r *CategoryRepository) GetBlogCategoryByID(ctx context.Context, id int64) (*domain.BlogCategory, error) {
	var category domain.BlogCategory
	query := `SELECT * FROM blog_categories WHERE id = $1`

	err := r.db.DB.GetContext(ctx, &category, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &category, err
}

func (r *CategoryRepository) UpdateBlogCategory(ctx context.Context, category *domain.BlogCategory) error {
	query := `
		UPDATE blog_categories 
		SET name = $1, description = $2, display_order = $3, is_active = $4, updated_at = NOW()
		WHERE id = $5
		RETURNING updated_at`

	return r.db.DB.QueryRowContext(
		ctx, query,
		category.Name, category.Description, category.DisplayOrder, category.IsActive, category.ID,
	).Scan(&category.UpdatedAt)
}

func (r *CategoryRepository) DeleteBlogCategory(ctx context.Context, id int64) error {
	query := `DELETE FROM blog_categories WHERE id = $1`
	_, err := r.db.DB.ExecContext(ctx, query, id)
	return err
}