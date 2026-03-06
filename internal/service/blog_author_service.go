package service

import (
	"context"
	"database/sql"

	"github.com/gosimple/slug"
	"github.com/merraki/merraki-backend/internal/domain"
	apperrors "github.com/merraki/merraki-backend/internal/pkg/errors"
	"github.com/merraki/merraki-backend/internal/repository/postgres"
)

type BlogAuthorService struct {
	authorRepo *postgres.BlogAuthorRepository
	logRepo    *postgres.ActivityLogRepository
}

func NewBlogAuthorService(
	authorRepo *postgres.BlogAuthorRepository,
	logRepo *postgres.ActivityLogRepository,
) *BlogAuthorService {
	return &BlogAuthorService{
		authorRepo: authorRepo,
		logRepo:    logRepo,
	}
}

func (s *BlogAuthorService) CreateAuthor(ctx context.Context, author *domain.BlogAuthor, createdBy int64) error {
	// Generate slug if not provided
	if author.Slug == "" {
		author.Slug = slug.Make(author.Name)
	}

	// Check if slug exists
	existing, err := s.authorRepo.FindBySlug(ctx, author.Slug)
	if err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to check slug", 500)
	}

	if existing != nil {
		return apperrors.New("SLUG_EXISTS", "Author with this slug already exists", 409)
	}

	if err := s.authorRepo.Create(ctx, author); err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to create author", 500)
	}

	_ = s.logRepo.Create(ctx, &domain.ActivityLog{
		AdminID:    &createdBy,
		Action:     "create_blog_author",
		EntityType: strPtr("blog_author"),
		EntityID:   &author.ID,
		Details:    domain.JSONMap{"name": author.Name},
	})

	return nil
}

func (s *BlogAuthorService) GetAuthorByID(ctx context.Context, id int64) (*domain.BlogAuthor, error) {
	author, err := s.authorRepo.FindByID(ctx, id)
	if err != nil {
		return nil, apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find author", 500)
	}

	if author == nil {
		return nil, apperrors.ErrNotFound
	}

	return author, nil
}

func (s *BlogAuthorService) GetAuthorBySlug(ctx context.Context, slug string) (*domain.BlogAuthor, error) {
	author, err := s.authorRepo.FindBySlug(ctx, slug)
	if err != nil {
		return nil, apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find author", 500)
	}

	if author == nil {
		return nil, apperrors.ErrNotFound
	}

	return author, nil
}

func (s *BlogAuthorService) GetAllAuthors(ctx context.Context, activeOnly bool, limit, offset int) ([]*domain.BlogAuthor, int, error) {
	return s.authorRepo.GetAll(ctx, activeOnly, limit, offset)
}

func (s *BlogAuthorService) UpdateAuthor(ctx context.Context, author *domain.BlogAuthor, updatedBy int64) error {
	existing, err := s.authorRepo.FindByID(ctx, author.ID)
	if err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find author", 500)
	}

	if existing == nil {
		return apperrors.ErrNotFound
	}

	// Check slug uniqueness if changed
	if author.Slug != existing.Slug {
		slugExists, err := s.authorRepo.FindBySlug(ctx, author.Slug)
		if err != nil && err != sql.ErrNoRows {
			return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to check slug", 500)
		}

		if slugExists != nil && slugExists.ID != author.ID {
			return apperrors.New("SLUG_EXISTS", "Author with this slug already exists", 409)
		}
	}

	if err := s.authorRepo.Update(ctx, author); err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to update author", 500)
	}

	_ = s.logRepo.Create(ctx, &domain.ActivityLog{
		AdminID:    &updatedBy,
		Action:     "update_blog_author",
		EntityType: strPtr("blog_author"),
		EntityID:   &author.ID,
		Details:    domain.JSONMap{"name": author.Name},
	})

	return nil
}

func (s *BlogAuthorService) DeleteAuthor(ctx context.Context, id, deletedBy int64) error {
	author, err := s.authorRepo.FindByID(ctx, id)
	if err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find author", 500)
	}

	if author == nil {
		return apperrors.ErrNotFound
	}

	if err := s.authorRepo.Delete(ctx, id); err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to delete author", 500)
	}

	_ = s.logRepo.Create(ctx, &domain.ActivityLog{
		AdminID:    &deletedBy,
		Action:     "delete_blog_author",
		EntityType: strPtr("blog_author"),
		EntityID:   &id,
		Details:    domain.JSONMap{"name": author.Name},
	})

	return nil
}
