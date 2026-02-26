package service

import (
	"context"
	"fmt"

	"github.com/gosimple/slug"
	"github.com/merraki/merraki-backend/internal/domain"
	apperrors "github.com/merraki/merraki-backend/internal/pkg/errors"
	"github.com/merraki/merraki-backend/internal/repository/postgres"
)

type TemplateService struct {
	templateRepo *postgres.TemplateRepository
	categoryRepo *postgres.CategoryRepository
	logRepo      *postgres.ActivityLogRepository
}

func NewTemplateService(
	templateRepo *postgres.TemplateRepository,
	categoryRepo *postgres.CategoryRepository,
	logRepo *postgres.ActivityLogRepository,
) *TemplateService {
	return &TemplateService{
		templateRepo: templateRepo,
		categoryRepo: categoryRepo,
		logRepo:      logRepo,
	}
}

func (s *TemplateService) CreateTemplate(ctx context.Context, template *domain.Template, createdBy int64) error {
	// Generate slug if not provided
	if template.Slug == "" {
		template.Slug = slug.Make(template.Title)
	}

	// Check if slug exists
	existing, err := s.templateRepo.FindBySlug(ctx, template.Slug)
	if err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to check slug", 500)
	}

	if existing != nil {
		return apperrors.New("SLUG_EXISTS", "Template with this slug already exists", 409)
	}

	// Verify category exists
	category, err := s.categoryRepo.GetTemplateCategoryByID(ctx, template.CategoryID)
	if err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find category", 500)
	}

	if category == nil {
		return apperrors.New("CATEGORY_NOT_FOUND", "Category not found", 404)
	}

	template.CreatedBy = &createdBy

	// Create template
	if err := s.templateRepo.Create(ctx, template); err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to create template", 500)
	}

	// Log activity
	_ = s.logRepo.Create(ctx, &domain.ActivityLog{
		AdminID:    &createdBy,
		Action:     "create_template",
		EntityType: strPtr("template"),
		EntityID:   &template.ID,
		Details:    domain.JSONMap{"title": template.Title, "slug": template.Slug},
	})

	return nil
}

func (s *TemplateService) GetTemplateByID(ctx context.Context, id int64) (*domain.Template, error) {
	template, err := s.templateRepo.FindByID(ctx, id)
	if err != nil {
		return nil, apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find template", 500)
	}

	if template == nil {
		return nil, apperrors.ErrNotFound
	}

	return template, nil
}

func (s *TemplateService) GetTemplateBySlug(ctx context.Context, slug string, incrementViews bool) (*domain.Template, error) {
	template, err := s.templateRepo.FindBySlug(ctx, slug)
	if err != nil {
		return nil, apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find template", 500)
	}

	if template == nil {
		return nil, apperrors.ErrNotFound
	}

	// Increment views for public access
	if incrementViews && template.Status == "active" {
		_ = s.templateRepo.IncrementViews(ctx, template.ID)
	}

	return template, nil
}

func (s *TemplateService) GetAllTemplates(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*domain.Template, int, error) {
	templates, total, err := s.templateRepo.GetAll(ctx, filters, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("repository error: %w", err)
	}
	return templates, total, nil
}

func (s *TemplateService) UpdateTemplate(ctx context.Context, template *domain.Template, updatedBy int64) error {
	// Check if template exists
	existing, err := s.templateRepo.FindByID(ctx, template.ID)
	if err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find template", 500)
	}

	if existing == nil {
		return apperrors.ErrNotFound
	}

	// Check if slug changed and is unique
	if template.Slug != existing.Slug {
		slugExists, err := s.templateRepo.FindBySlug(ctx, template.Slug)
		if err != nil {
			return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to check slug", 500)
		}

		if slugExists != nil {
			return apperrors.New("SLUG_EXISTS", "Template with this slug already exists", 409)
		}
	}

	// Update
	if err := s.templateRepo.Update(ctx, template); err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to update template", 500)
	}

	// Log activity
	_ = s.logRepo.Create(ctx, &domain.ActivityLog{
		AdminID:    &updatedBy,
		Action:     "update_template",
		EntityType: strPtr("template"),
		EntityID:   &template.ID,
		Details:    domain.JSONMap{"title": template.Title},
	})

	return nil
}

func (s *TemplateService) DeleteTemplate(ctx context.Context, id, deletedBy int64) error {
	// Check if template exists
	template, err := s.templateRepo.FindByID(ctx, id)
	if err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find template", 500)
	}

	if template == nil {
		return apperrors.ErrNotFound
	}

	// Hard delete from database
	if err := s.templateRepo.Delete(ctx, id); err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to delete template", 500)
	}

	// Log activity
	_ = s.logRepo.Create(ctx, &domain.ActivityLog{
		AdminID:    &deletedBy,
		Action:     "delete_template",
		EntityType: strPtr("template"),
		EntityID:   &id,
		Details:    domain.JSONMap{"title": template.Title, "id": id, "permanent": true},
	})

	return nil
}

func (s *TemplateService) GetFeaturedTemplates(ctx context.Context, limit int) ([]*domain.Template, error) {
	return s.templateRepo.GetFeatured(ctx, limit)
}

func (s *TemplateService) GetPopularTemplates(ctx context.Context, limit int) ([]*domain.Template, error) {
	return s.templateRepo.GetPopular(ctx, limit)
}

func (s *TemplateService) SearchTemplates(ctx context.Context, query string, limit int) ([]*domain.Template, error) {
	if query == "" {
		return []*domain.Template{}, nil
	}
	return s.templateRepo.Search(ctx, query, limit)
}

func (s *TemplateService) GetTemplatesByIDs(ctx context.Context, ids []int64) ([]*domain.Template, error) {
	return s.templateRepo.GetByIDs(ctx, ids)
}

func (s *TemplateService) GetAnalytics(ctx context.Context) (map[string]interface{}, error) {
	analytics, err := s.templateRepo.GetAnalytics(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository error: %w", err)
	}
	return analytics, nil
}