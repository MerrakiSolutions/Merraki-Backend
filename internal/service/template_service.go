package service

import (
	"context"
	"database/sql"
	"time"

	"github.com/gosimple/slug"
	"github.com/lib/pq"
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
		template.Slug = slug.Make(template.Name)
	}

	// Check if slug exists
	existing, err := s.templateRepo.FindBySlug(ctx, template.Slug)
	if err != nil && err != sql.ErrNoRows {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to check slug", 500)
	}
	if existing != nil {
		return apperrors.New("SLUG_EXISTS", "Template with this slug already exists", 409)
	}

	// Verify category exists
	if template.CategoryID != nil {
		category, err := s.categoryRepo.FindByID(ctx, *template.CategoryID)
		if err != nil && err != sql.ErrNoRows {
			return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to verify category", 500)
		}
		if category == nil {
			return apperrors.New("CATEGORY_NOT_FOUND", "Category not found", 404)
		}
	}

	// Set published_at if available
	if template.IsAvailable && template.PublishedAt == nil {
		now := time.Now()
		template.PublishedAt = &now
	}

	// Initialize arrays
	if template.MetaKeywords == nil {
		template.MetaKeywords = pq.StringArray{}
	}

	if err := s.templateRepo.Create(ctx, template); err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to create template", 500)
	}

	_ = s.logRepo.Create(ctx, &domain.ActivityLog{
		AdminID:    &createdBy,
		Action:     "create_template",
		EntityType: strPtr("template"),
		EntityID:   &template.ID,
		Details:    domain.JSONMap{"name": template.Name},
	})

	return nil
}

func (s *TemplateService) GetTemplateByID(ctx context.Context, id int64, incrementViews bool) (*domain.Template, error) {
	template, err := s.templateRepo.FindByID(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, apperrors.ErrNotFound
		}
		return nil, apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find template", 500)
	}
	if template == nil {
		return nil, apperrors.ErrNotFound
	}

	if incrementViews {
		_ = s.templateRepo.IncrementViews(ctx, id)
	}

	return template, nil
}

func (s *TemplateService) GetTemplateBySlug(ctx context.Context, slug string, incrementViews bool) (*domain.Template, error) {
	template, err := s.templateRepo.FindBySlug(ctx, slug)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, apperrors.ErrNotFound
		}
		return nil, apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find template", 500)
	}
	if template == nil {
		return nil, apperrors.ErrNotFound
	}

	if incrementViews {
		_ = s.templateRepo.IncrementViews(ctx, template.ID)
	}

	return template, nil
}

func (s *TemplateService) GetTemplateByName(ctx context.Context, name string) (*domain.Template, error) {
	template, err := s.templateRepo.FindByName(ctx, name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, apperrors.ErrNotFound
		}
		return nil, apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find template", 500)
	}
	if template == nil {
		return nil, apperrors.ErrNotFound
	}
	return template, nil
}

func (s *TemplateService) GetAllTemplates(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*domain.Template, int, error) {
	return s.templateRepo.GetAll(ctx, filters, limit, offset)
}

func (s *TemplateService) GetAllTemplatesWithRelations(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*domain.TemplateWithRelations, int, error) {
	return s.templateRepo.GetAllWithRelations(ctx, filters, limit, offset)
}

func (s *TemplateService) UpdateTemplate(ctx context.Context, template *domain.Template, updatedBy int64) error {
	existing, err := s.templateRepo.FindByID(ctx, template.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return apperrors.ErrNotFound
		}
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find template", 500)
	}
	if existing == nil {
		return apperrors.ErrNotFound
	}

	// Check slug uniqueness if changed
	if template.Slug != existing.Slug {
		slugExists, err := s.templateRepo.FindBySlug(ctx, template.Slug)
		if err != nil && err != sql.ErrNoRows {
			return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to check slug", 500)
		}
		if slugExists != nil && slugExists.ID != template.ID {
			return apperrors.New("SLUG_EXISTS", "Template with this slug already exists", 409)
		}
	}

	// Verify category if provided
	if template.CategoryID != nil {
		category, err := s.categoryRepo.FindByID(ctx, *template.CategoryID)
		if err != nil && err != sql.ErrNoRows {
			return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to verify category", 500)
		}
		if category == nil {
			return apperrors.New("CATEGORY_NOT_FOUND", "Category not found", 404)
		}
	}

	// Set published_at if becoming available
	if template.IsAvailable && !existing.IsAvailable && template.PublishedAt == nil {
		now := time.Now()
		template.PublishedAt = &now
	}

	// Initialize arrays
	if template.MetaKeywords == nil {
		template.MetaKeywords = pq.StringArray{}
	}

	if err := s.templateRepo.Update(ctx, template); err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to update template", 500)
	}

	_ = s.logRepo.Create(ctx, &domain.ActivityLog{
		AdminID:    &updatedBy,
		Action:     "update_template",
		EntityType: strPtr("template"),
		EntityID:   &template.ID,
		Details:    domain.JSONMap{"name": template.Name},
	})

	return nil
}

func (s *TemplateService) PatchTemplate(ctx context.Context, id int64, updates map[string]interface{}, updatedBy int64) error {
	existing, err := s.templateRepo.FindByID(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return apperrors.ErrNotFound
		}
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find template", 500)
	}
	if existing == nil {
		return apperrors.ErrNotFound
	}

	// Handle availability change
	if isAvailable, ok := updates["is_available"].(bool); ok && isAvailable && !existing.IsAvailable {
		if _, hasPublishedAt := updates["published_at"]; !hasPublishedAt {
			updates["published_at"] = time.Now()
		}
	}

	if err := s.templateRepo.Patch(ctx, id, updates); err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to patch template", 500)
	}

	_ = s.logRepo.Create(ctx, &domain.ActivityLog{
		AdminID:    &updatedBy,
		Action:     "patch_template",
		EntityType: strPtr("template"),
		EntityID:   &id,
		Details:    domain.JSONMap{"updates": updates},
	})

	return nil
}

func (s *TemplateService) DeleteTemplate(ctx context.Context, id, deletedBy int64) error {
	template, err := s.templateRepo.FindByID(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return apperrors.ErrNotFound
		}
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find template", 500)
	}
	if template == nil {
		return apperrors.ErrNotFound
	}

	if err := s.templateRepo.Delete(ctx, id); err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to delete template", 500)
	}

	_ = s.logRepo.Create(ctx, &domain.ActivityLog{
		AdminID:    &deletedBy,
		Action:     "delete_template",
		EntityType: strPtr("template"),
		EntityID:   &id,
		Details:    domain.JSONMap{"name": template.Name},
	})

	return nil
}

func (s *TemplateService) SearchTemplates(ctx context.Context, query string, limit int) ([]*domain.Template, error) {
	return s.templateRepo.Search(ctx, query, limit)
}

func (s *TemplateService) GetTemplatesByCategory(ctx context.Context, categorySlug string, limit, offset int) ([]*domain.Template, int, error) {
	category, err := s.categoryRepo.FindBySlug(ctx, categorySlug)
	if err != nil {
		return nil, 0, apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find category", 500)
	}
	if category == nil {
		return nil, 0, apperrors.ErrNotFound
	}

	return s.templateRepo.GetByCategory(ctx, category.ID, limit, offset)
}

func (s *TemplateService) GetTemplatesByTag(ctx context.Context, tag string, limit, offset int) ([]*domain.Template, int, error) {
	return s.templateRepo.GetByTag(ctx, tag, limit, offset)
}

func (s *TemplateService) GetFeaturedTemplates(ctx context.Context, limit int) ([]*domain.Template, error) {
	return s.templateRepo.GetFeatured(ctx, limit)
}

func (s *TemplateService) GetBestsellers(ctx context.Context, limit int) ([]*domain.Template, error) {
	return s.templateRepo.GetBestsellers(ctx, limit)
}

func (s *TemplateService) GetNewTemplates(ctx context.Context, limit int) ([]*domain.Template, error) {
	return s.templateRepo.GetNew(ctx, limit)
}

// Stock management
func (s *TemplateService) UpdateStock(ctx context.Context, id int64, quantity int, updatedBy int64) error {
	updates := map[string]interface{}{
		"stock_quantity": quantity,
	}
	return s.PatchTemplate(ctx, id, updates, updatedBy)
}

func (s *TemplateService) DecrementStock(ctx context.Context, id int64, quantity int) error {
	return s.templateRepo.DecrementStock(ctx, id, quantity)
}

// Images
func (s *TemplateService) AddImage(ctx context.Context, image *domain.TemplateImage, createdBy int64) error {
	if err := s.templateRepo.CreateImage(ctx, image); err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to add image", 500)
	}

	_ = s.logRepo.Create(ctx, &domain.ActivityLog{
		AdminID:    &createdBy,
		Action:     "add_template_image",
		EntityType: strPtr("template_image"),
		EntityID:   &image.ID,
		Details:    domain.JSONMap{"template_id": image.TemplateID},
	})

	return nil
}

func (s *TemplateService) DeleteImage(ctx context.Context, id, deletedBy int64) error {
	if err := s.templateRepo.DeleteImage(ctx, id); err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to delete image", 500)
	}

	_ = s.logRepo.Create(ctx, &domain.ActivityLog{
		AdminID:    &deletedBy,
		Action:     "delete_template_image",
		EntityType: strPtr("template_image"),
		EntityID:   &id,
	})

	return nil
}

func (s *TemplateService) GetImages(ctx context.Context, templateID int64) ([]*domain.TemplateImage, error) {
	return s.templateRepo.GetImages(ctx, templateID)
}

// Features
func (s *TemplateService) AddFeature(ctx context.Context, feature *domain.TemplateFeature, createdBy int64) error {
	if err := s.templateRepo.CreateFeature(ctx, feature); err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to add feature", 500)
	}

	_ = s.logRepo.Create(ctx, &domain.ActivityLog{
		AdminID:    &createdBy,
		Action:     "add_template_feature",
		EntityType: strPtr("template_feature"),
		EntityID:   &feature.ID,
		Details:    domain.JSONMap{"template_id": feature.TemplateID},
	})

	return nil
}

func (s *TemplateService) DeleteFeature(ctx context.Context, id, deletedBy int64) error {
	if err := s.templateRepo.DeleteFeature(ctx, id); err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to delete feature", 500)
	}

	_ = s.logRepo.Create(ctx, &domain.ActivityLog{
		AdminID:    &deletedBy,
		Action:     "delete_template_feature",
		EntityType: strPtr("template_feature"),
		EntityID:   &id,
	})

	return nil
}

func (s *TemplateService) GetFeatures(ctx context.Context, templateID int64) ([]*domain.TemplateFeature, error) {
	return s.templateRepo.GetFeatures(ctx, templateID)
}

// Tags
func (s *TemplateService) UpdateTags(ctx context.Context, templateID int64, tags []string, updatedBy int64) error {
	if err := s.templateRepo.ReplaceAllTags(ctx, templateID, tags); err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to update tags", 500)
	}

	_ = s.logRepo.Create(ctx, &domain.ActivityLog{
		AdminID:    &updatedBy,
		Action:     "update_template_tags",
		EntityType: strPtr("template"),
		EntityID:   &templateID,
		Details:    domain.JSONMap{"tags": tags},
	})

	return nil
}

func (s *TemplateService) GetTags(ctx context.Context, templateID int64) ([]string, error) {
	return s.templateRepo.GetTags(ctx, templateID)
}

// Analytics
func (s *TemplateService) TrackEvent(ctx context.Context, event *domain.TemplateAnalytics) error {
	return s.templateRepo.CreateAnalyticsEvent(ctx, event)
}