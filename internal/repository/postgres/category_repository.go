package postgres

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/merraki/merraki-backend/internal/domain"
)

type CategoryRepository struct {
	db *sqlx.DB
}

func NewCategoryRepository(db *sqlx.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

func (r *CategoryRepository) Create(ctx context.Context, category *domain.Category) error {
	query := `
		INSERT INTO categories (
			name, slug, description, parent_id, display_order, 
			is_active, meta_title, meta_description
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`

	return r.db.QueryRowContext(
		ctx, query,
		category.Name,
		category.Slug,
		category.Description,
		category.ParentID,
		category.DisplayOrder,
		category.IsActive,
		category.MetaTitle,
		category.MetaDescription,
	).Scan(&category.ID, &category.CreatedAt, &category.UpdatedAt)
}

func (r *CategoryRepository) FindByID(ctx context.Context, id int64) (*domain.Category, error) {
	var category domain.Category
	query := `SELECT * FROM categories WHERE id = $1`

	err := r.db.GetContext(ctx, &category, query, id)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return &category, err
}

func (r *CategoryRepository) FindBySlug(ctx context.Context, slug string) (*domain.Category, error) {
	var category domain.Category
	query := `SELECT * FROM categories WHERE slug = $1`

	err := r.db.GetContext(ctx, &category, query, slug)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return &category, err
}

func (r *CategoryRepository) GetAll(ctx context.Context, activeOnly bool) ([]*domain.Category, error) {
	var categories []*domain.Category
	
	query := `SELECT * FROM categories`
	if activeOnly {
		query += ` WHERE is_active = true`
	}
	query += ` ORDER BY display_order ASC, name ASC`

	err := r.db.SelectContext(ctx, &categories, query)
	return categories, err
}

func (r *CategoryRepository) Update(ctx context.Context, category *domain.Category) error {
	query := `
		UPDATE categories SET
			name = $1,
			slug = $2,
			description = $3,
			parent_id = $4,
			display_order = $5,
			is_active = $6,
			meta_title = $7,
			meta_description = $8
		WHERE id = $9
	`

	result, err := r.db.ExecContext(
		ctx, query,
		category.Name,
		category.Slug,
		category.Description,
		category.ParentID,
		category.DisplayOrder,
		category.IsActive,
		category.MetaTitle,
		category.MetaDescription,
		category.ID,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (r *CategoryRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM categories WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrNotFound
	}

	return nil
}