package postgres

import (
	"context"
	"database/sql"

	"github.com/merraki/merraki-backend/internal/domain"
)

type BlogAuthorRepository struct {
	db *Database
}

func NewBlogAuthorRepository(db *Database) *BlogAuthorRepository {
	return &BlogAuthorRepository{db: db}
}

func (r *BlogAuthorRepository) Create(ctx context.Context, author *domain.BlogAuthor) error {
	query := `
		INSERT INTO blog_authors (admin_id, name, slug, email, bio, avatar_url, social_links, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at`

	return r.db.DB.QueryRowContext(
		ctx, query,
		author.AdminID, author.Name, author.Slug, author.Email,
		author.Bio, author.AvatarURL, author.SocialLinks, author.IsActive,
	).Scan(&author.ID, &author.CreatedAt, &author.UpdatedAt)
}

func (r *BlogAuthorRepository) FindByID(ctx context.Context, id int64) (*domain.BlogAuthor, error) {
	var author domain.BlogAuthor
	query := `SELECT * FROM blog_authors WHERE id = $1`

	err := r.db.DB.GetContext(ctx, &author, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &author, err
}

func (r *BlogAuthorRepository) FindBySlug(ctx context.Context, slug string) (*domain.BlogAuthor, error) {
	var author domain.BlogAuthor
	query := `SELECT * FROM blog_authors WHERE slug = $1`

	err := r.db.DB.GetContext(ctx, &author, query, slug)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &author, err
}

func (r *BlogAuthorRepository) GetAll(ctx context.Context, activeOnly bool, limit, offset int) ([]*domain.BlogAuthor, int, error) {
	var authors []*domain.BlogAuthor
	
	query := `SELECT * FROM blog_authors WHERE 1=1`
	countQuery := `SELECT COUNT(*) FROM blog_authors WHERE 1=1`

	if activeOnly {
		query += " AND is_active = true"
		countQuery += " AND is_active = true"
	}

	query += " ORDER BY name ASC LIMIT $1 OFFSET $2"

	var total int
	if err := r.db.DB.GetContext(ctx, &total, countQuery); err != nil {
		return nil, 0, err
	}

	if err := r.db.DB.SelectContext(ctx, &authors, query, limit, offset); err != nil {
		return nil, 0, err
	}

	return authors, total, nil
}

func (r *BlogAuthorRepository) Update(ctx context.Context, author *domain.BlogAuthor) error {
	query := `
		UPDATE blog_authors 
		SET name = $1, slug = $2, email = $3, bio = $4, avatar_url = $5,
		    social_links = $6, is_active = $7, updated_at = NOW()
		WHERE id = $8
		RETURNING updated_at`

	return r.db.DB.QueryRowContext(
		ctx, query,
		author.Name, author.Slug, author.Email, author.Bio,
		author.AvatarURL, author.SocialLinks, author.IsActive, author.ID,
	).Scan(&author.UpdatedAt)
}

func (r *BlogAuthorRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM blog_authors WHERE id = $1`
	_, err := r.db.DB.ExecContext(ctx, query, id)
	return err
}