package service

import (
	"context"

	"github.com/gosimple/slug"
	"github.com/merraki/merraki-backend/internal/domain"
	apperrors "github.com/merraki/merraki-backend/internal/pkg/errors"
	"github.com/merraki/merraki-backend/internal/repository/postgres"
	"github.com/valyala/fasthttp"
)

type CategoryService struct {
	categoryRepo *postgres.CategoryRepository
	logRepo      *postgres.ActivityLogRepository
}

func (s *CategoryService) GetTemplateCategoryBySlug(ctx *fasthttp.RequestCtx, categorySlug string) (any, any) {
	panic("unimplemented")
}

func NewCategoryService(
	categoryRepo *postgres.CategoryRepository,
	logRepo *postgres.ActivityLogRepository,
) *CategoryService {
	return &CategoryService{
		categoryRepo: categoryRepo,
		logRepo:      logRepo,
	}
}

// Template Categories
func (s *CategoryService) CreateTemplateCategory(ctx context.Context, category *domain.TemplateCategory, createdBy int64) error {
	if category.Slug == "" {
		category.Slug = slug.Make(category.Name)
	}

	// Check if slug exists
	existing, err := s.categoryRepo.GetTemplateCategoryBySlug(ctx, category.Slug)
	if err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to check slug", 500)
	}

	if existing != nil {
		return apperrors.New("SLUG_EXISTS", "Category with this slug already exists", 409)
	}

	if err := s.categoryRepo.CreateTemplateCategory(ctx, category); err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to create category", 500)
	}

	_ = s.logRepo.Create(ctx, &domain.ActivityLog{
		AdminID:    &createdBy,
		Action:     "create_template_category",
		EntityType: strPtr("template_category"),
		EntityID:   &category.ID,
		Details:    domain.JSONMap{"name": category.Name},
	})

	return nil
}

func (s *CategoryService) GetTemplateCategories(ctx context.Context, activeOnly bool) ([]*domain.TemplateCategory, error) {
	return s.categoryRepo.GetTemplateCategories(ctx, activeOnly)
}

func (s *CategoryService) GetTemplateCategoryByID(ctx context.Context, id int64) (*domain.TemplateCategory, error) {
	category, err := s.categoryRepo.GetTemplateCategoryByID(ctx, id)
	if err != nil {
		return nil, apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find category", 500)
	}

	if category == nil {
		return nil, apperrors.ErrNotFound
	}

	return category, nil
}

func (s *CategoryService) UpdateTemplateCategory(ctx context.Context, category *domain.TemplateCategory, updatedBy int64) error {
	existing, err := s.categoryRepo.GetTemplateCategoryByID(ctx, category.ID)
	if err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find category", 500)
	}

	if existing == nil {
		return apperrors.ErrNotFound
	}

	if err := s.categoryRepo.UpdateTemplateCategory(ctx, category); err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to update category", 500)
	}

	_ = s.logRepo.Create(ctx, &domain.ActivityLog{
		AdminID:    &updatedBy,
		Action:     "update_template_category",
		EntityType: strPtr("template_category"),
		EntityID:   &category.ID,
		Details:    domain.JSONMap{"name": category.Name},
	})

	return nil
}

func (s *CategoryService) DeleteTemplateCategory(ctx context.Context, id, deletedBy int64) error {
	category, err := s.categoryRepo.GetTemplateCategoryByID(ctx, id)
	if err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find category", 500)
	}

	if category == nil {
		return apperrors.ErrNotFound
	}

	// Check if category has templates
	if category.TemplatesCount > 0 {
		return apperrors.New("CATEGORY_HAS_TEMPLATES", "Cannot delete category with templates", 400)
	}

	if err := s.categoryRepo.DeleteTemplateCategory(ctx, id); err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to delete category", 500)
	}

	_ = s.logRepo.Create(ctx, &domain.ActivityLog{
		AdminID:    &deletedBy,
		Action:     "delete_template_category",
		EntityType: strPtr("template_category"),
		EntityID:   &id,
		Details:    domain.JSONMap{"name": category.Name},
	})

	return nil
}

// Blog Categories (similar pattern)
func (s *CategoryService) CreateBlogCategory(ctx context.Context, category *domain.BlogCategory, createdBy int64) error {
	if category.Slug == "" {
		category.Slug = slug.Make(category.Name)
	}

	if err := s.categoryRepo.CreateBlogCategory(ctx, category); err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to create category", 500)
	}

	return nil
}

func (s *CategoryService) GetBlogCategories(ctx context.Context, activeOnly bool) ([]*domain.BlogCategory, error) {
	return s.categoryRepo.GetBlogCategories(ctx, activeOnly)
}

func (s *CategoryService) UpdateBlogCategory(ctx context.Context, category *domain.BlogCategory) error {
	return s.categoryRepo.UpdateBlogCategory(ctx, category)
}

func (s *CategoryService) DeleteBlogCategory(ctx context.Context, id int64) error {
	return s.categoryRepo.DeleteBlogCategory(ctx, id)
}
