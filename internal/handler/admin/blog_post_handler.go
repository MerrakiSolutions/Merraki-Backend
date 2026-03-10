package admin

import (
	"strconv"
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

type BlogPostHandler struct {
	postService *service.BlogPostService
}

func NewBlogPostHandler(postService *service.BlogPostService) *BlogPostHandler {
	return &BlogPostHandler{postService: postService}
}

type CreateBlogPostRequest struct {
	Title              string   `json:"title" validate:"required"`
	Slug               string   `json:"slug"`
	Excerpt            string   `json:"excerpt"`
	Content            string   `json:"content" validate:"required"`
	FeaturedImageURL   string   `json:"featured_image_url"`
	AuthorID           *int64   `json:"author_id"`
	CategoryID         *int64   `json:"category_id"`
	Tags               []string `json:"tags"`
	MetaTitle          string   `json:"meta_title"`
	MetaDescription    string   `json:"meta_description"`
	MetaKeywords       []string `json:"meta_keywords"`
	Status             string   `json:"status" validate:"required,oneof=draft published archived"`
	IsFeatured         bool     `json:"is_featured"`
	ReadingTimeMinutes int      `json:"reading_time_minutes"`
}

// ✅ FIX: Add UpdateBlogPostRequest struct
type UpdateBlogPostRequest struct {
	Title              string   `json:"title" validate:"required"`
	Slug               string   `json:"slug"`
	Excerpt            string   `json:"excerpt"`
	Content            string   `json:"content" validate:"required"`
	FeaturedImageURL   string   `json:"featured_image_url"`
	AuthorID           *int64   `json:"author_id"`
	CategoryID         *int64   `json:"category_id"`
	Tags               []string `json:"tags"`
	MetaTitle          string   `json:"meta_title"`
	MetaDescription    string   `json:"meta_description"`
	MetaKeywords       []string `json:"meta_keywords"`
	Status             string   `json:"status" validate:"required,oneof=draft published archived"`
	IsFeatured         bool     `json:"is_featured"`
	ReadingTimeMinutes int      `json:"reading_time_minutes"`
}

func (h *BlogPostHandler) GetAll(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	withRelations := c.Query("with_relations") == "true"

	params := &domain.PaginationParams{
		Page:  page,
		Limit: limit,
	}
	params.Validate()

	filters := make(map[string]interface{})

	// Status filter
	if status := c.Query("status"); status != "" {
		filters["status"] = status
	}

	// Author filter
	if authorID := c.Query("author_id"); authorID != "" {
		if id, err := strconv.ParseInt(authorID, 10, 64); err == nil {
			filters["author_id"] = id
		}
	}

	// Category filter
	if categoryID := c.Query("category_id"); categoryID != "" {
		if id, err := strconv.ParseInt(categoryID, 10, 64); err == nil {
			filters["category_id"] = id
		}
	}

	// Tag filter
	if tag := c.Query("tag"); tag != "" {
		filters["tag"] = tag
	}

	// Search filter
	if search := c.Query("search"); search != "" {
		filters["search"] = search
	}

	// Featured filter
	if featured := c.Query("featured"); featured != "" {
		filters["featured"] = featured == "true"
	}

	// Date range
	if startDate := c.Query("start_date"); startDate != "" {
		filters["start_date"] = startDate
	}
	if endDate := c.Query("end_date"); endDate != "" {
		filters["end_date"] = endDate
	}

	// Sort
	if sort := c.Query("sort"); sort != "" {
		filters["sort"] = sort
	}

	if withRelations {
		posts, total, err := h.postService.GetAllPostsWithRelations(c.Context(), filters, params.Limit, params.GetOffset())
		if err != nil {
			logger.Error("Failed to get posts with relations", zap.Error(err))
			return response.Error(c, err)
		}
		return response.Paginated(c, posts, total, params.Page, params.Limit)
	}

	posts, total, err := h.postService.GetAllPosts(c.Context(), filters, params.Limit, params.GetOffset())
	if err != nil {
		logger.Error("Failed to get posts", zap.Error(err))
		return response.Error(c, err)
	}

	return response.Paginated(c, posts, total, params.Page, params.Limit)
}

func (h *BlogPostHandler) GetByID(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, fiber.NewError(400, "Invalid post ID"))
	}

	post, err := h.postService.GetPostByID(c.Context(), int64(id), false)
	if err != nil {
		return response.Error(c, err)
	}

	return response.SuccessData(c, post)
}

func (h *BlogPostHandler) GetBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")

	post, err := h.postService.GetPostBySlug(c.Context(), slug, false)
	if err != nil {
		return response.Error(c, err)
	}

	return response.SuccessData(c, post)
}

func (h *BlogPostHandler) Create(c *fiber.Ctx) error {
	var req CreateBlogPostRequest
	if err := c.BodyParser(&req); err != nil {
		logger.Error("Failed to parse blog post request", zap.Error(err))
		return response.Error(c, fiber.NewError(400, "Invalid request body"))
	}

	if err := validator.Validate(req); err != nil {
		return response.ValidationError(c, validator.FormatValidationErrors(err))
	}

	adminID := middleware.GetAdminID(c)

	var publishedAt *time.Time
	if req.Status == "published" {
		now := time.Now()
		publishedAt = &now
	}

	// Ensure arrays are not nil
	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}

	metaKeywords := req.MetaKeywords
	if metaKeywords == nil {
		metaKeywords = []string{}
	}

	// Handle optional string fields
	var excerpt *string
	if req.Excerpt != "" {
		excerpt = &req.Excerpt
	}

	var featuredImageURL *string
	if req.FeaturedImageURL != "" {
		featuredImageURL = &req.FeaturedImageURL
	}

	var metaTitle *string
	if req.MetaTitle != "" {
		metaTitle = &req.MetaTitle
	}

	var metaDescription *string
	if req.MetaDescription != "" {
		metaDescription = &req.MetaDescription
	}

	var readingTime *int
	if req.ReadingTimeMinutes > 0 {
		readingTime = &req.ReadingTimeMinutes
	}

	post := &domain.BlogPost{
		Title:              req.Title,
		Slug:               req.Slug,
		Excerpt:            excerpt,
		Content:            req.Content,
		FeaturedImageURL:   featuredImageURL,
		AuthorID:           req.AuthorID,
		CategoryID:         req.CategoryID,
		Tags:               pq.StringArray(tags),
		MetaTitle:          metaTitle,
		MetaDescription:    metaDescription,
		MetaKeywords:       pq.StringArray(metaKeywords),
		Status:             req.Status,
		IsFeatured:         req.IsFeatured,
		ReadingTimeMinutes: readingTime,
		PublishedAt:        publishedAt,
	}

	if err := h.postService.CreatePost(c.Context(), post, adminID); err != nil {
		logger.Error("Failed to create blog post", zap.Error(err))
		return response.Error(c, err)
	}

	return response.Created(c, "Blog post created successfully", post)
}

// ✅ FIX: Complete Update handler
func (h *BlogPostHandler) Update(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.NewError(fiber.StatusBadRequest, "Invalid post ID"))
	}

	var req UpdateBlogPostRequest
	if err := c.BodyParser(&req); err != nil {
		logger.Error("Failed to parse update request", zap.Error(err))
		return response.Error(c, fiber.NewError(400, "Invalid request body"))
	}

	if err := validator.Validate(req); err != nil {
		return response.ValidationError(c, validator.FormatValidationErrors(err))
	}

	adminID := middleware.GetAdminID(c)

	// ✅ Get existing post
	existingPost, err := h.postService.GetPostByID(c.Context(), int64(id), false)
	if err != nil {
		return response.Error(c, err)
	}

	// ✅ Ensure arrays are not nil
	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}

	metaKeywords := req.MetaKeywords
	if metaKeywords == nil {
		metaKeywords = []string{}
	}

	// Handle optional string fields
	var excerpt *string
	if req.Excerpt != "" {
		excerpt = &req.Excerpt
	}

	var featuredImageURL *string
	if req.FeaturedImageURL != "" {
		featuredImageURL = &req.FeaturedImageURL
	}

	var metaTitle *string
	if req.MetaTitle != "" {
		metaTitle = &req.MetaTitle
	}

	var metaDescription *string
	if req.MetaDescription != "" {
		metaDescription = &req.MetaDescription
	}

	var readingTime *int
	if req.ReadingTimeMinutes > 0 {
		readingTime = &req.ReadingTimeMinutes
	}

	// ✅ Build updated post
	post := &domain.BlogPost{
		ID:                 int64(id),
		Title:              req.Title,
		Slug:               req.Slug,
		Excerpt:            excerpt,
		Content:            req.Content,
		FeaturedImageURL:   featuredImageURL,
		AuthorID:           req.AuthorID,
		CategoryID:         req.CategoryID,
		Tags:               pq.StringArray(tags),
		MetaTitle:          metaTitle,
		MetaDescription:    metaDescription,
		MetaKeywords:       pq.StringArray(metaKeywords),
		Status:             req.Status,
		IsFeatured:         req.IsFeatured,
		ReadingTimeMinutes: readingTime,
	}

	// ✅ Handle published_at
	if req.Status == "published" && existingPost.Status != "published" {
		now := time.Now()
		post.PublishedAt = &now
	} else if req.Status == "published" && existingPost.PublishedAt != nil {
		post.PublishedAt = existingPost.PublishedAt
	} else if req.Status != "published" {
		post.PublishedAt = nil
	}

	if err := h.postService.UpdatePost(c.Context(), post, adminID); err != nil {
		logger.Error("Failed to update blog post", zap.Error(err))
		return response.Error(c, err)
	}

	return response.Success(c, "Blog post updated successfully", post)
}

// Patch - Partial update (PATCH method)
func (h *BlogPostHandler) Patch(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, fiber.NewError(400, "Invalid post ID"))
	}

	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return response.Error(c, err)
	}

	adminID := middleware.GetAdminID(c)

	if err := h.postService.PatchPost(c.Context(), int64(id), updates, adminID); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, "Blog post updated successfully", nil)
}

func (h *BlogPostHandler) Delete(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, fiber.NewError(400, "Invalid post ID"))
	}

	adminID := middleware.GetAdminID(c)

	if err := h.postService.DeletePost(c.Context(), int64(id), adminID); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, "Blog post deleted successfully", nil)
}

func (h *BlogPostHandler) Search(c *fiber.Ctx) error {
	query := c.Query("q")
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	if query == "" {
		return response.Error(c, fiber.NewError(400, "Search query required"))
	}

	posts, err := h.postService.SearchPosts(c.Context(), query, limit)
	if err != nil {
		return response.Error(c, err)
	}

	return response.SuccessData(c, fiber.Map{
		"results": posts,
		"query":   query,
		"total":   len(posts),
	})
}

// Helper function
func strPtr(s string) *string {
	return &s
}