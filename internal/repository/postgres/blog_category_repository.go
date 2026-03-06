package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/merraki/merraki-backend/internal/domain"
)

type BlogCategoryRepository struct {
	db *Database
}

func NewBlogCategoryRepository(db *Database) *BlogCategoryRepository {
	return &BlogCategoryRepository{db: db}
}

func (r *BlogCategoryRepository) Create(ctx context.Context, category *domain.BlogCategory) error {
	query := `
		INSERT INTO blog_categories (name, slug, description, parent_id, display_order, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`

	return r.db.DB.QueryRowContext(
		ctx, query,
		category.Name, category.Slug, category.Description,
		category.ParentID, category.DisplayOrder, category.IsActive,
	).Scan(&category.ID, &category.CreatedAt, &category.UpdatedAt)
}

func (r *BlogCategoryRepository) FindByID(ctx context.Context, id int64) (*domain.BlogCategory, error) {
	var category domain.BlogCategory
	query := `SELECT * FROM blog_categories WHERE id = $1`

	err := r.db.DB.GetContext(ctx, &category, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &category, err
}

func (r *BlogCategoryRepository) FindBySlug(ctx context.Context, slug string) (*domain.BlogCategory, error) {
	var category domain.BlogCategory
	query := `SELECT * FROM blog_categories WHERE slug = $1`

	err := r.db.DB.GetContext(ctx, &category, query, slug)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &category, err
}

func (r *BlogCategoryRepository) GetAll(ctx context.Context, activeOnly bool, limit, offset int) ([]*domain.BlogCategory, int, error) {
	var categories []*domain.BlogCategory
	
	query := `SELECT * FROM blog_categories WHERE 1=1`
	countQuery := `SELECT COUNT(*) FROM blog_categories WHERE 1=1`

	if activeOnly {
		query += " AND is_active = true"
		countQuery += " AND is_active = true"
	}

	query += " ORDER BY display_order ASC, name ASC LIMIT $1 OFFSET $2"

	var total int
	if err := r.db.DB.GetContext(ctx, &total, countQuery); err != nil {
		return nil, 0, err
	}

	if err := r.db.DB.SelectContext(ctx, &categories, query, limit, offset); err != nil {
		return nil, 0, err
	}

	return categories, total, nil
}

func (r *BlogCategoryRepository) Update(ctx context.Context, category *domain.BlogCategory) error {
	query := `
		UPDATE blog_categories 
		SET name = $1, slug = $2, description = $3, parent_id = $4,
		    display_order = $5, is_active = $6, updated_at = NOW()
		WHERE id = $7
		RETURNING updated_at`

	return r.db.DB.QueryRowContext(
		ctx, query,
		category.Name, category.Slug, category.Description,
		category.ParentID, category.DisplayOrder, category.IsActive, category.ID,
	).Scan(&category.UpdatedAt)
}

func (r *BlogCategoryRepository) Delete(ctx context.Context, id int64) error {
	// Check if any posts use this category
	var count int
	checkQuery := `SELECT COUNT(*) FROM blog_posts WHERE category_id = $1`
	if err := r.db.DB.GetContext(ctx, &count, checkQuery, id); err != nil {
		return err
	}

	if count > 0 {
		return fmt.Errorf("cannot delete category: %d blog posts are using it", count)
	}

	query := `DELETE FROM blog_categories WHERE id = $1`
	_, err := r.db.DB.ExecContext(ctx, query, id)
	return err
}