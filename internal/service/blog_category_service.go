package service

import (
	"context"
	"database/sql"

	"github.com/gosimple/slug"
	"github.com/merraki/merraki-backend/internal/domain"
	apperrors "github.com/merraki/merraki-backend/internal/pkg/errors"
	"github.com/merraki/merraki-backend/internal/repository/postgres"
)

type BlogCategoryService struct {
	categoryRepo *postgres.BlogCategoryRepository
	logRepo      *postgres.ActivityLogRepository
}

func NewBlogCategoryService(
	categoryRepo *postgres.BlogCategoryRepository,
	logRepo *postgres.ActivityLogRepository,
) *BlogCategoryService {
	return &BlogCategoryService{
		categoryRepo: categoryRepo,
		logRepo:      logRepo,
	}
}

func (s *BlogCategoryService) CreateCategory(ctx context.Context, category *domain.BlogCategory, createdBy int64) error {
	// Generate slug if not provided
	if category.Slug == "" {
		category.Slug = slug.Make(category.Name)
	}

	// Check if slug exists
	existing, err := s.categoryRepo.FindBySlug(ctx, category.Slug)
	if err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to check slug", 500)
	}

	if existing != nil {
		return apperrors.New("SLUG_EXISTS", "Category with this slug already exists", 409)
	}

	if err := s.categoryRepo.Create(ctx, category); err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to create category", 500)
	}

	_ = s.logRepo.Create(ctx, &domain.ActivityLog{
		AdminID:    &createdBy,
		Action:     "create_blog_category",
		EntityType: strPtr("blog_category"),
		EntityID:   &category.ID,
		Details:    domain.JSONMap{"name": category.Name},
	})

	return nil
}

func (s *BlogCategoryService) GetCategoryByID(ctx context.Context, id int64) (*domain.BlogCategory, error) {
	category, err := s.categoryRepo.FindByID(ctx, id)
	if err != nil {
		return nil, apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find category", 500)
	}

	if category == nil {
		return nil, apperrors.ErrNotFound
	}

	return category, nil
}

func (s *BlogCategoryService) GetCategoryBySlug(ctx context.Context, slug string) (*domain.BlogCategory, error) {
	category, err := s.categoryRepo.FindBySlug(ctx, slug)
	if err != nil {
		return nil, apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find category", 500)
	}

	if category == nil {
		return nil, apperrors.ErrNotFound
	}

	return category, nil
}

func (s *BlogCategoryService) GetAllCategories(ctx context.Context, activeOnly bool, limit, offset int) ([]*domain.BlogCategory, int, error) {
	return s.categoryRepo.GetAll(ctx, activeOnly, limit, offset)
}

func (s *BlogCategoryService) UpdateCategory(ctx context.Context, category *domain.BlogCategory, updatedBy int64) error {
	existing, err := s.categoryRepo.FindByID(ctx, category.ID)
	if err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find category", 500)
	}

	if existing == nil {
		return apperrors.ErrNotFound
	}

	// Check slug uniqueness if changed
	if category.Slug != existing.Slug {
		slugExists, err := s.categoryRepo.FindBySlug(ctx, category.Slug)
		if err != nil && err != sql.ErrNoRows {
			return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to check slug", 500)
		}

		if slugExists != nil && slugExists.ID != category.ID {
			return apperrors.New("SLUG_EXISTS", "Category with this slug already exists", 409)
		}
	}

	if err := s.categoryRepo.Update(ctx, category); err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to update category", 500)
	}

	_ = s.logRepo.Create(ctx, &domain.ActivityLog{
		AdminID:    &updatedBy,
		Action:     "update_blog_category",
		EntityType: strPtr("blog_category"),
		EntityID:   &category.ID,
		Details:    domain.JSONMap{"name": category.Name},
	})

	return nil
}

func (s *BlogCategoryService) DeleteCategory(ctx context.Context, id, deletedBy int64) error {
	category, err := s.categoryRepo.FindByID(ctx, id)
	if err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find category", 500)
	}

	if category == nil {
		return apperrors.ErrNotFound
	}

	if err := s.categoryRepo.Delete(ctx, id); err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to delete category", 500)
	}

	_ = s.logRepo.Create(ctx, &domain.ActivityLog{
		AdminID:    &deletedBy,
		Action:     "delete_blog_category",
		EntityType: strPtr("blog_category"),
		EntityID:   &id,
		Details:    domain.JSONMap{"name": category.Name},
	})

	return nil
}