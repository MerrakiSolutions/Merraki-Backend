package admin

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/lib/pq"
	"github.com/merraki/merraki-backend/internal/domain"
	"github.com/merraki/merraki-backend/internal/middleware"
	"github.com/merraki/merraki-backend/internal/pkg/logger"
	"github.com/merraki/merraki-backend/internal/pkg/response"
	"github.com/merraki/merraki-backend/internal/pkg/validator"
	"github.com/merraki/merraki-backend/internal/service"
	"go.uber.org/zap"
)

type BlogHandler struct {
	blogService *service.BlogService
}

func NewBlogHandler(blogService *service.BlogService) *BlogHandler {
	return &BlogHandler{blogService: blogService}
}

func (h *BlogHandler) GetAll(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	params := &domain.PaginationParams{
		Page:  page,
		Limit: limit,
	}
	params.Validate()

	filters := make(map[string]interface{})
	if status := c.Query("status"); status != "" {
		filters["status"] = status
	}
	if categoryID := c.Query("category_id"); categoryID != "" {
		if id, err := strconv.ParseInt(categoryID, 10, 64); err == nil {
			filters["category_id"] = id
		}
	}
	if search := c.Query("search"); search != "" {
		filters["search"] = search
	}

	posts, total, err := h.blogService.GetAllPosts(c.Context(), filters, params.Limit, params.GetOffset())
	if err != nil {
		return response.Error(c, err)
	}

	return response.Paginated(c, posts, total, params.Page, params.Limit)
}

func (h *BlogHandler) GetByID(c *fiber.Ctx) error {
	_, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, fiber.NewError(400, "Invalid post ID"))
	}

	post, err := h.blogService.GetPostBySlug(c.Context(), "", false)
	if err != nil {
		return response.Error(c, err)
	}

	return response.SuccessData(c, post)
}

type CreateBlogPostRequest struct {
	Slug               string   `json:"slug" validate:"slug"`
	Title              string   `json:"title" validate:"required"`
	Excerpt            string   `json:"excerpt"`
	Content            string   `json:"content" validate:"required"`
	FeaturedImageURL   string   `json:"cover_image"`
	CategoryID         *int64   `json:"category_id"`
	Tags               []string `json:"tags"`
	Status             string   `json:"status" validate:"required,oneof=draft published"`
	IsFeatured         bool     `json:"is_featured"`
	MetaTitle          string   `json:"meta_title"`
	MetaDescription    string   `json:"meta_description"`
	SEOKeywords        []string `json:"seo_keywords"`
	ReadingTimeMinutes int      `json:"reading_time_minutes"`
}

func (h *BlogHandler) Create(c *fiber.Ctx) error {
	var req CreateBlogPostRequest
	if err := c.BodyParser(&req); err != nil {
		logger.Error("Failed to parse blog post request", 
			zap.Error(err),
			zap.String("body", string(c.Body())),
		)
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
			"code":    "INVALID_REQUEST",
		})
	}

	if err := validator.Validate(req); err != nil {
		logger.Error("Validation failed", 
			zap.Error(err),
			zap.Any("request", req),
		)
		return response.ValidationError(c, validator.FormatValidationErrors(err))
	}

	adminID := middleware.GetAdminID(c)
	logger.Info("Creating blog post", 
		zap.String("title", req.Title),
		zap.Int64("admin_id", adminID),
	)

	var publishedAt *time.Time
	if req.Status == "published" {
		now := time.Now()
		publishedAt = &now
	}

	// Calculate reading time
	if req.ReadingTimeMinutes == 0 {
		wordCount := len(strings.Fields(req.Content))
		req.ReadingTimeMinutes = (wordCount / 200) + 1
	}

	// Ensure tags are not nil
	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}
	
	seoKeywords := req.SEOKeywords
	if seoKeywords == nil {
		seoKeywords = []string{}
	}

	post := &domain.BlogPost{
		Slug:               req.Slug,
		Title:              req.Title,
		Excerpt:            &req.Excerpt,
		Content:            req.Content,
		FeaturedImageURL:   &req.FeaturedImageURL,
		CategoryID:         req.CategoryID,
		Tags:               pq.StringArray(tags),
		Status:             req.Status,
		MetaTitle:          &req.MetaTitle,
		MetaDescription:    &req.MetaDescription,
		SEOKeywords:        pq.StringArray(seoKeywords),
		ReadingTimeMinutes: &req.ReadingTimeMinutes,
		AuthorID:           &adminID,
		PublishedAt:        publishedAt,
	}

	if err := h.blogService.CreatePost(c.Context(), post, adminID); err != nil {
		logger.Error("Failed to create blog post", 
			zap.Error(err),
			zap.String("error_type", fmt.Sprintf("%T", err)),
			zap.String("title", req.Title),
		)
		
		// Show detailed error in development
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),  // Show actual error
			"code":    "DATABASE_ERROR",
		})
	}

	logger.Info("Blog post created successfully", zap.Int64("post_id", post.ID))
	return response.Created(c, "Blog post created successfully", post)
}

func (h *BlogHandler) Update(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		logger.Error("Invalid post ID", zap.Error(err))
		return response.Error(c, fiber.NewError(400, "Invalid post ID"))
	}

	var req CreateBlogPostRequest
	if err := c.BodyParser(&req); err != nil {
		logger.Error("Failed to parse blog post update request", 
			zap.Error(err),
			zap.String("body", string(c.Body())),
		)
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
			"code":    "INVALID_REQUEST",
		})
	}

	if err := validator.Validate(req); err != nil {
		logger.Error("Validation failed", zap.Error(err))
		return response.ValidationError(c, validator.FormatValidationErrors(err))
	}

	adminID := middleware.GetAdminID(c)
	logger.Info("Updating blog post", 
		zap.Int("post_id", id),
		zap.Int64("admin_id", adminID),
	)

	// Handle published_at based on status
	var publishedAt *time.Time
	if req.Status == "published" {
		now := time.Now()
		publishedAt = &now
	}

	// Calculate reading time if not provided
	if req.ReadingTimeMinutes == 0 {
		wordCount := len(strings.Fields(req.Content))
		req.ReadingTimeMinutes = (wordCount / 200) + 1
	}

	// Ensure arrays are not nil
	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}
	
	seoKeywords := req.SEOKeywords
	if seoKeywords == nil {
		seoKeywords = []string{}
	}

	post := &domain.BlogPost{
		ID:                 int64(id),
		Slug:               req.Slug,
		Title:              req.Title,
		Excerpt:            &req.Excerpt,
		Content:            req.Content,
		FeaturedImageURL:   &req.FeaturedImageURL,
		CategoryID:         req.CategoryID,
		Tags:               pq.StringArray(tags),
		Status:             req.Status,
		MetaTitle:          &req.MetaTitle,
		MetaDescription:    &req.MetaDescription,
		SEOKeywords:        pq.StringArray(seoKeywords),
		ReadingTimeMinutes: &req.ReadingTimeMinutes,
		PublishedAt:        publishedAt,
	}

	if err := h.blogService.UpdatePost(c.Context(), post, adminID); err != nil {
		logger.Error("Failed to update blog post", 
			zap.Error(err),
			zap.Int("post_id", id),
		)
		
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
			"code":    "DATABASE_ERROR",
		})
	}

	logger.Info("Blog post updated successfully", zap.Int("post_id", id))
	return response.Success(c, "Blog post updated successfully", post)
}

func (h *BlogHandler) Delete(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, fiber.NewError(400, "Invalid post ID"))
	}

	adminID := middleware.GetAdminID(c)

	if err := h.blogService.DeletePost(c.Context(), int64(id), adminID); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, "Blog post deleted successfully", nil)
}

func (h *BlogHandler) GetAnalytics(c *fiber.Ctx) error {
	analytics, err := h.blogService.GetAnalytics(c.Context())
	if err != nil {
		return response.Error(c, err)
	}

	return response.SuccessData(c, analytics)
}
