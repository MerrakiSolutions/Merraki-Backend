package public

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/merraki/merraki-backend/internal/domain"
	"github.com/merraki/merraki-backend/internal/pkg/response"
	"github.com/merraki/merraki-backend/internal/service"
)

type BlogHandler struct {
	postService     *service.BlogPostService
	authorService   *service.BlogAuthorService
	categoryService *service.BlogCategoryService
}

func NewBlogHandler(
	postService *service.BlogPostService,
	authorService *service.BlogAuthorService,
	categoryService *service.BlogCategoryService,
) *BlogHandler {
	return &BlogHandler{
		postService:     postService,
		authorService:   authorService,
		categoryService: categoryService,
	}
}

// ========== POSTS ==========

func (h *BlogHandler) GetAllPosts(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "12"))
	withRelations := c.Query("with_relations") == "true"

	params := &domain.PaginationParams{
		Page:  page,
		Limit: limit,
	}
	params.Validate()

	filters := map[string]interface{}{
		"status": "published", // Only published posts for public
	}

	// Category filter
	if categorySlug := c.Query("category"); categorySlug != "" {
		category, err := h.categoryService.GetCategoryBySlug(c.Context(), categorySlug)
		if err == nil && category != nil {
			filters["category_id"] = category.ID
		}
	}

	// Author filter
	if authorSlug := c.Query("author"); authorSlug != "" {
		author, err := h.authorService.GetAuthorBySlug(c.Context(), authorSlug)
		if err == nil && author != nil {
			filters["author_id"] = author.ID
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

	// Sort
	sort := c.Query("sort", "newest")
	filters["sort"] = sort

	if withRelations {
		posts, total, err := h.postService.GetAllPostsWithRelations(c.Context(), filters, params.Limit, params.GetOffset())
		if err != nil {
			return response.Error(c, err)
		}
		return response.Paginated(c, posts, total, params.Page, params.Limit)
	}

	posts, total, err := h.postService.GetAllPosts(c.Context(), filters, params.Limit, params.GetOffset())
	if err != nil {
		return response.Error(c, err)
	}

	return response.Paginated(c, posts, total, params.Page, params.Limit)
}

func (h *BlogHandler) GetPostBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	incrementViews := c.Query("increment_views", "true") == "true"

	post, err := h.postService.GetPostBySlug(c.Context(), slug, incrementViews)
	if err != nil {
		return response.Error(c, err)
	}

	// Only show published posts to public
	if post.Status != "published" {
		return response.Error(c, fiber.NewError(404, "Post not found"))
	}

	return response.SuccessData(c, post)
}

func (h *BlogHandler) SearchPosts(c *fiber.Ctx) error {
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

// ========== AUTHORS ==========

func (h *BlogHandler) GetAllAuthors(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	params := &domain.PaginationParams{
		Page:  page,
		Limit: limit,
	}
	params.Validate()

	authors, total, err := h.authorService.GetAllAuthors(c.Context(), true, params.Limit, params.GetOffset())
	if err != nil {
		return response.Error(c, err)
	}

	return response.Paginated(c, authors, total, params.Page, params.Limit)
}

func (h *BlogHandler) GetAuthorBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")

	author, err := h.authorService.GetAuthorBySlug(c.Context(), slug)
	if err != nil {
		return response.Error(c, err)
	}

	if !author.IsActive {
		return response.Error(c, fiber.NewError(404, "Author not found"))
	}

	return response.SuccessData(c, author)
}

func (h *BlogHandler) GetPostsByAuthor(c *fiber.Ctx) error {
	slug := c.Params("slug")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "12"))

	params := &domain.PaginationParams{
		Page:  page,
		Limit: limit,
	}
	params.Validate()

	author, err := h.authorService.GetAuthorBySlug(c.Context(), slug)
	if err != nil {
		return response.Error(c, err)
	}

	posts, total, err := h.postService.GetPostsByAuthor(c.Context(), author.ID, params.Limit, params.GetOffset())
	if err != nil {
		return response.Error(c, err)
	}

	return response.Paginated(c, posts, total, params.Page, params.Limit)
}

// ========== CATEGORIES ==========

func (h *BlogHandler) GetAllCategories(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))

	params := &domain.PaginationParams{
		Page:  page,
		Limit: limit,
	}
	params.Validate()

	categories, total, err := h.categoryService.GetAllCategories(c.Context(), true, params.Limit, params.GetOffset())
	if err != nil {
		return response.Error(c, err)
	}

	return response.Paginated(c, categories, total, params.Page, params.Limit)
}

func (h *BlogHandler) GetCategoryBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")

	category, err := h.categoryService.GetCategoryBySlug(c.Context(), slug)
	if err != nil {
		return response.Error(c, err)
	}

	if !category.IsActive {
		return response.Error(c, fiber.NewError(404, "Category not found"))
	}

	return response.SuccessData(c, category)
}

func (h *BlogHandler) GetPostsByCategory(c *fiber.Ctx) error {
	slug := c.Params("slug")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "12"))

	params := &domain.PaginationParams{
		Page:  page,
		Limit: limit,
	}
	params.Validate()

	category, err := h.categoryService.GetCategoryBySlug(c.Context(), slug)
	if err != nil {
		return response.Error(c, err)
	}

	posts, total, err := h.postService.GetPostsByCategory(c.Context(), category.ID, params.Limit, params.GetOffset())
	if err != nil {
		return response.Error(c, err)
	}

	return response.Paginated(c, posts, total, params.Page, params.Limit)
}

// ========== TAGS ==========

func (h *BlogHandler) GetPostsByTag(c *fiber.Ctx) error {
	tag := c.Params("tag")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "12"))

	params := &domain.PaginationParams{
		Page:  page,
		Limit: limit,
	}
	params.Validate()

	posts, total, err := h.postService.GetPostsByTag(c.Context(), tag, params.Limit, params.GetOffset())
	if err != nil {
		return response.Error(c, err)
	}

	return response.Paginated(c, posts, total, params.Page, params.Limit)
}