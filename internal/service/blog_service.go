package service

import (
	"context"
	"database/sql"

	"github.com/gosimple/slug"
	"github.com/merraki/merraki-backend/internal/domain"
	apperrors "github.com/merraki/merraki-backend/internal/pkg/errors"
	"github.com/merraki/merraki-backend/internal/repository/postgres"
)

type BlogService struct {
	blogRepo *postgres.BlogRepository
	logRepo  *postgres.ActivityLogRepository
}

func NewBlogService(
	blogRepo *postgres.BlogRepository,
	logRepo *postgres.ActivityLogRepository,
) *BlogService {
	return &BlogService{
		blogRepo: blogRepo,
		logRepo:  logRepo,
	}
}

func (s *BlogService) CreatePost(ctx context.Context, post *domain.BlogPost, createdBy int64) error {
	if post.Slug == "" {
		post.Slug = slug.Make(post.Title)
	}

	// Check if slug exists
	existing, err := s.blogRepo.FindBySlug(ctx, post.Slug)
	if err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to check slug", 500)
	}

	if existing != nil {
		return apperrors.New("SLUG_EXISTS", "Post with this slug already exists", 409)
	}

	if err := s.blogRepo.Create(ctx, post); err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to create post", 500)
	}

	_ = s.logRepo.Create(ctx, &domain.ActivityLog{
		AdminID:    &createdBy,
		Action:     "create_blog_post",
		EntityType: strPtr("blog_post"),
		EntityID:   &post.ID,
		Details:    domain.JSONMap{"title": post.Title},
	})

	return nil
}

func (s *BlogService) GetPostBySlug(ctx context.Context, slug string, incrementViews bool) (*domain.BlogPost, error) {
	post, err := s.blogRepo.FindBySlug(ctx, slug)
	if err != nil {
		return nil, apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find post", 500)
	}

	if post == nil {
		return nil, apperrors.ErrNotFound
	}

	if incrementViews && post.Status == "published" {
		_ = s.blogRepo.IncrementViews(ctx, post.ID)
	}

	return post, nil
}

func (s *BlogService) GetPostByID(ctx context.Context, id int64) (*domain.BlogPost, error) {
	post, err := s.blogRepo.FindByID(ctx, id)
	if err != nil {
		return nil, apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find post", 500)
	}

	if post == nil {
		return nil, apperrors.ErrNotFound
	}

	return post, nil
}

func (s *BlogService) GetAllPosts(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*domain.BlogPost, int, error) {
	return s.blogRepo.GetAll(ctx, filters, limit, offset)
}

func (s *BlogService) UpdatePost(ctx context.Context, post *domain.BlogPost, updatedBy int64) error {
	// Check if post exists
	existing, err := s.blogRepo.FindByID(ctx, post.ID)
	if err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find post", 500)
	}

	if existing == nil {
		return apperrors.ErrNotFound
	}

	// Check if slug changed and is unique
	if post.Slug != existing.Slug {
		slugExists, err := s.blogRepo.FindBySlug(ctx, post.Slug)
		if err != nil && err != sql.ErrNoRows {
			return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to check slug", 500)
		}

		if slugExists != nil && slugExists.ID != post.ID {
			return apperrors.New("SLUG_EXISTS", "Post with this slug already exists", 409)
		}
	}

	// Update
	if err := s.blogRepo.Update(ctx, post); err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to update post", 500)
	}

	// Log activity
	_ = s.logRepo.Create(ctx, &domain.ActivityLog{
		AdminID:    &updatedBy,
		Action:     "update_blog_post",
		EntityType: strPtr("blog_post"),
		EntityID:   &post.ID,
		Details:    domain.JSONMap{"title": post.Title, "status": post.Status},
	})

	return nil
}

func (s *BlogService) DeletePost(ctx context.Context, id, deletedBy int64) error {
	post, err := s.blogRepo.FindByID(ctx, id)
	if err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find post", 500)
	}

	if post == nil {
		return apperrors.ErrNotFound
	}

	if err := s.blogRepo.Delete(ctx, id); err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to delete post", 500)
	}

	_ = s.logRepo.Create(ctx, &domain.ActivityLog{
		AdminID:    &deletedBy,
		Action:     "delete_blog_post",
		EntityType: strPtr("blog_post"),
		EntityID:   &id,
		Details:    domain.JSONMap{"title": post.Title},
	})

	return nil
}

func (s *BlogService) GetFeaturedPosts(ctx context.Context, limit int) ([]*domain.BlogPost, error) {
	return s.blogRepo.GetFeatured(ctx, limit)
}

func (s *BlogService) GetPopularPosts(ctx context.Context, limit int) ([]*domain.BlogPost, error) {
	return s.blogRepo.GetPopular(ctx, limit)
}

func (s *BlogService) SearchPosts(ctx context.Context, query string, limit int) ([]*domain.BlogPost, error) {
	return s.blogRepo.Search(ctx, query, limit)
}

func (s *BlogService) GetAnalytics(ctx context.Context) (map[string]interface{}, error) {
	return s.blogRepo.GetAnalytics(ctx)
}