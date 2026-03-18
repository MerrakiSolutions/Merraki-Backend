package service

import (
	"context"

	"github.com/gosimple/slug"
	"github.com/merraki/merraki-backend/internal/domain"
	"github.com/merraki/merraki-backend/internal/pkg/logger"
	"github.com/merraki/merraki-backend/internal/repository"
	"go.uber.org/zap"
)

type CategoryService struct {
	categoryRepo    repository.CategoryRepository
	activityLogRepo repository.ActivityLogRepository
}

func NewCategoryService(
	categoryRepo repository.CategoryRepository,
	activityLogRepo repository.ActivityLogRepository,
) *CategoryService {
	return &CategoryService{
		categoryRepo:    categoryRepo,
		activityLogRepo: activityLogRepo,
	}
}

func (s *CategoryService) CreateCategory(ctx context.Context, category *domain.Category, createdBy int64) error {
	// Generate slug if not provided
	if category.Slug == "" {
		category.Slug = slug.Make(category.Name)
	}

	// Check if slug exists
	existing, err := s.categoryRepo.FindBySlug(ctx, category.Slug)
	if err != nil && err != domain.ErrNotFound {
		return err
	}
	if existing != nil {
		return domain.ErrDuplicateEntry
	}

	// Create category
	if err := s.categoryRepo.Create(ctx, category); err != nil {
		return err
	}

	// Log activity
	s.logActivity(ctx, "create_category", category.ID, createdBy, map[string]interface{}{
		"name": category.Name,
		"slug": category.Slug,
	})

	logger.Info("Category created",
		zap.Int64("id", category.ID),
		zap.String("name", category.Name),
	)

	return nil
}

func (s *CategoryService) GetCategoryByID(ctx context.Context, id int64) (*domain.Category, error) {
	return s.categoryRepo.FindByID(ctx, id)
}

func (s *CategoryService) GetCategoryBySlug(ctx context.Context, slug string) (*domain.Category, error) {
	return s.categoryRepo.FindBySlug(ctx, slug)
}

func (s *CategoryService) GetAllCategories(ctx context.Context, activeOnly bool) ([]*domain.Category, error) {
	return s.categoryRepo.GetAll(ctx, activeOnly)
}

func (s *CategoryService) UpdateCategory(ctx context.Context, category *domain.Category, updatedBy int64) error {
	// Check if exists
	existing, err := s.categoryRepo.FindByID(ctx, category.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return domain.ErrNotFound
	}

	// Check slug uniqueness if changed
	if category.Slug != existing.Slug {
		slugExists, err := s.categoryRepo.FindBySlug(ctx, category.Slug)
		if err != nil && err != domain.ErrNotFound {
			return err
		}
		if slugExists != nil && slugExists.ID != category.ID {
			return domain.ErrDuplicateEntry
		}
	}

	// Update
	if err := s.categoryRepo.Update(ctx, category); err != nil {
		return err
	}

	// Log activity
	s.logActivity(ctx, "update_category", category.ID, updatedBy, map[string]interface{}{
		"name": category.Name,
	})

	logger.Info("Category updated",
		zap.Int64("id", category.ID),
		zap.String("name", category.Name),
	)

	return nil
}

func (s *CategoryService) DeleteCategory(ctx context.Context, id int64, deletedBy int64) error {
	// Check if exists
	category, err := s.categoryRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if category == nil {
		return domain.ErrNotFound
	}

	// Delete
	if err := s.categoryRepo.Delete(ctx, id); err != nil {
		return err
	}

	// Log activity
	s.logActivity(ctx, "delete_category", id, deletedBy, map[string]interface{}{
		"name": category.Name,
	})

	logger.Info("Category deleted",
		zap.Int64("id", id),
		zap.String("name", category.Name),
	)

	return nil
}

func (s *CategoryService) logActivity(ctx context.Context, action string, entityID int64, adminID int64, metadata map[string]interface{}) {
	if s.activityLogRepo == nil {
		return
	}

	jsonMetadata := make(domain.JSONMap)
	for k, v := range metadata {
		jsonMetadata[k] = v
	}

	entityType := "category"
	var adminIDPtr *int64
	if adminID > 0 {
		adminIDPtr = &adminID
	}

	activity := &domain.ActivityLog{
		Action:     action,
		EntityType: &entityType,
		EntityID:   &entityID,
		AdminID:    adminIDPtr,
		Details:    jsonMetadata,
	}

	_ = s.activityLogRepo.Create(ctx, activity)
}