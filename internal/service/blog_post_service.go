package service

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/gosimple/slug"
	"github.com/lib/pq"
	"github.com/merraki/merraki-backend/internal/domain"
	apperrors "github.com/merraki/merraki-backend/internal/pkg/errors"
	"github.com/merraki/merraki-backend/internal/repository/postgres"
)

type BlogPostService struct {
	postRepo     *postgres.BlogPostRepository
	authorRepo   *postgres.BlogAuthorRepository
	categoryRepo *postgres.BlogCategoryRepository
	logRepo      *postgres.ActivityLogRepository
}

func NewBlogPostService(
	postRepo *postgres.BlogPostRepository,
	authorRepo *postgres.BlogAuthorRepository,
	categoryRepo *postgres.BlogCategoryRepository,
	logRepo *postgres.ActivityLogRepository,
) *BlogPostService {
	return &BlogPostService{
		postRepo:     postRepo,
		authorRepo:   authorRepo,
		categoryRepo: categoryRepo,
		logRepo:      logRepo,
	}
}

func (s *BlogPostService) CreatePost(ctx context.Context, post *domain.BlogPost, createdBy int64) error {
	// Generate slug if not provided
	if post.Slug == "" {
		post.Slug = slug.Make(post.Title)
	}

	// Check if slug exists
	existing, err := s.postRepo.FindBySlug(ctx, post.Slug)
	if err != nil && err != sql.ErrNoRows {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to check slug", 500)
	}

	if existing != nil {
		return apperrors.New("SLUG_EXISTS", "Post with this slug already exists", 409)
	}

	// Verify author exists
	if post.AuthorID != nil {
		author, err := s.authorRepo.FindByID(ctx, *post.AuthorID)
		if err != nil && err != sql.ErrNoRows {
			return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to verify author", 500)
		}
		if author == nil {
			return apperrors.New("AUTHOR_NOT_FOUND", "Author not found", 404)
		}
	}

	// Verify category exists
	if post.CategoryID != nil {
		category, err := s.categoryRepo.FindByID(ctx, *post.CategoryID)
		if err != nil && err != sql.ErrNoRows {
			return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to verify category", 500)
		}
		if category == nil {
			return apperrors.New("CATEGORY_NOT_FOUND", "Category not found", 404)
		}
	}

	// Calculate reading time if not provided
	if post.ReadingTimeMinutes == nil || *post.ReadingTimeMinutes == 0 {
		wordCount := len(strings.Fields(post.Content))
		readingTime := (wordCount / 200) + 1
		post.ReadingTimeMinutes = &readingTime
	}

	// Set published_at if status is published
	if post.Status == "published" && post.PublishedAt == nil {
		now := time.Now()
		post.PublishedAt = &now
	}

	// Ensure arrays are initialized
	if post.Tags == nil {
		post.Tags = pq.StringArray{}
	}
	if post.MetaKeywords == nil {
		post.MetaKeywords = pq.StringArray{}
	}

	if err := s.postRepo.Create(ctx, post); err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to create post", 500)
	}

	_ = s.logRepo.Create(ctx, &domain.ActivityLog{
		AdminID:    &createdBy,
		Action:     "create_blog_post",
		EntityType: strPtr("blog_post"),
		EntityID:   &post.ID,
		Details:    domain.JSONMap{"title": post.Title, "status": post.Status},
	})

	return nil
}

func (s *BlogPostService) GetPostByID(ctx context.Context, id int64, incrementViews bool) (*domain.BlogPost, error) {
	post, err := s.postRepo.FindByID(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, apperrors.ErrNotFound
		}
		return nil, apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find post", 500)
	}

	if post == nil {
		return nil, apperrors.ErrNotFound
	}

	if incrementViews {
		_ = s.postRepo.IncrementViews(ctx, id)
	}

	return post, nil
}

func (s *BlogPostService) GetPostBySlug(ctx context.Context, slug string, incrementViews bool) (*domain.BlogPost, error) {
	post, err := s.postRepo.FindBySlug(ctx, slug)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, apperrors.ErrNotFound
		}
		return nil, apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find post", 500)
	}

	if post == nil {
		return nil, apperrors.ErrNotFound
	}

	if incrementViews {
		_ = s.postRepo.IncrementViews(ctx, post.ID)
	}

	return post, nil
}

func (s *BlogPostService) GetAllPosts(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*domain.BlogPost, int, error) {
	return s.postRepo.GetAll(ctx, filters, limit, offset)
}

func (s *BlogPostService) GetAllPostsWithRelations(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*domain.BlogPostWithRelations, int, error) {
	return s.postRepo.GetAllWithRelations(ctx, filters, limit, offset)
}

// ✅ FIX: Complete UpdatePost method
func (s *BlogPostService) UpdatePost(ctx context.Context, post *domain.BlogPost, updatedBy int64) error {
	existing, err := s.postRepo.FindByID(ctx, post.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return apperrors.ErrNotFound
		}
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find post", 500)
	}

	if existing == nil {
		return apperrors.ErrNotFound
	}

	// Check slug uniqueness if changed
	if post.Slug != existing.Slug {
		slugExists, err := s.postRepo.FindBySlug(ctx, post.Slug)
		if err != nil && err != sql.ErrNoRows {
			return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to check slug", 500)
		}

		if slugExists != nil && slugExists.ID != post.ID {
			return apperrors.New("SLUG_EXISTS", "Post with this slug already exists", 409)
		}
	}

	// Verify author if provided
	if post.AuthorID != nil {
		author, err := s.authorRepo.FindByID(ctx, *post.AuthorID)
		if err != nil && err != sql.ErrNoRows {
			return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to verify author", 500)
		}
		if author == nil {
			return apperrors.New("AUTHOR_NOT_FOUND", "Author not found", 404)
		}
	}

	// Verify category if provided
	if post.CategoryID != nil {
		category, err := s.categoryRepo.FindByID(ctx, *post.CategoryID)
		if err != nil && err != sql.ErrNoRows {
			return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to verify category", 500)
		}
		if category == nil {
			return apperrors.New("CATEGORY_NOT_FOUND", "Category not found", 404)
		}
	}

	// ✅ Ensure arrays are initialized
	if post.Tags == nil {
		post.Tags = pq.StringArray{}
	}
	if post.MetaKeywords == nil {
		post.MetaKeywords = pq.StringArray{}
	}

	// Set published_at if status changed to published
	if post.Status == "published" && existing.Status != "published" && post.PublishedAt == nil {
		now := time.Now()
		post.PublishedAt = &now
	}

	if err := s.postRepo.Update(ctx, post); err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to update post", 500)
	}

	_ = s.logRepo.Create(ctx, &domain.ActivityLog{
		AdminID:    &updatedBy,
		Action:     "update_blog_post",
		EntityType: strPtr("blog_post"),
		EntityID:   &post.ID,
		Details:    domain.JSONMap{"title": post.Title, "status": post.Status},
	})

	return nil
}

// PatchPost - Partial update (PATCH support)
func (s *BlogPostService) PatchPost(ctx context.Context, id int64, updates map[string]interface{}, updatedBy int64) error {
	existing, err := s.postRepo.FindByID(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return apperrors.ErrNotFound
		}
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find post", 500)
	}

	if existing == nil {
		return apperrors.ErrNotFound
	}

	// Handle status change to published
	if status, ok := updates["status"].(string); ok && status == "published" && existing.Status != "published" {
		if _, hasPublishedAt := updates["published_at"]; !hasPublishedAt {
			updates["published_at"] = time.Now()
		}
	}

	if err := s.postRepo.Patch(ctx, id, updates); err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to patch post", 500)
	}

	_ = s.logRepo.Create(ctx, &domain.ActivityLog{
		AdminID:    &updatedBy,
		Action:     "patch_blog_post",
		EntityType: strPtr("blog_post"),
		EntityID:   &id,
		Details:    domain.JSONMap{"updates": updates},
	})

	return nil
}

func (s *BlogPostService) DeletePost(ctx context.Context, id, deletedBy int64) error {
	post, err := s.postRepo.FindByID(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return apperrors.ErrNotFound
		}
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find post", 500)
	}

	if post == nil {
		return apperrors.ErrNotFound
	}

	if err := s.postRepo.Delete(ctx, id); err != nil {
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

func (s *BlogPostService) SearchPosts(ctx context.Context, query string, limit int) ([]*domain.BlogPost, error) {
	return s.postRepo.Search(ctx, query, limit)
}

func (s *BlogPostService) GetPostsByAuthor(
	ctx context.Context, 
	authorID int64, 
	limit, 
	offset int) ([]*domain.BlogPost, int, error) {
	// Get author by ID first
	author, err := s.authorRepo.FindByID(ctx, authorID)
	if err != nil {
		return nil, 0, apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find author", 500)
	}
	if author == nil {
		return nil, 0, apperrors.ErrNotFound
	}

	return s.postRepo.GetByAuthor(ctx, authorID, limit, offset)
}

func (s *BlogPostService) GetPostsByCategory(
    ctx context.Context,
    categoryID int64,
    limit int,
    offset int,
) ([]*domain.BlogPost, int, error) {
    return s.postRepo.GetByCategory(ctx, categoryID, limit, offset)
}

func (s *BlogPostService) GetPostsByTag(ctx context.Context, tag string, limit, offset int) ([]*domain.BlogPost, int, error) {
	return s.postRepo.GetByTag(ctx, tag, limit, offset)
}
