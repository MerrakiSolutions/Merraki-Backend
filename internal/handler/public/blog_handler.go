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

// ── Posts ─────────────────────────────────────────────────────────────────────

func (h *BlogHandler) GetAllPosts(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "12"))
	category := c.Query("category")
	tag := c.Query("tag")
	search := c.Query("search")
	featured := c.Query("featured") == "true"

	params := &domain.PaginationParams{
		Page:  page,
		Limit: limit,
	}
	params.Validate()

	// Build filters
	filters := map[string]interface{}{
		"status": "published", // Only published posts for public
	}
	if category != "" {
		filters["category"] = category
	}
	if tag != "" {
		filters["tag"] = tag
	}
	if search != "" {
		filters["search"] = search
	}
	if featured {
		filters["featured"] = true
	}

	posts, total, err := h.postService.GetAllPostsWithRelations(
		c.Context(),
		filters,
		params.Limit,
		params.GetOffset(),
	)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Paginated(c, posts, total, params.Page, params.Limit)
}

func (h *BlogHandler) GetPostBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")

	post, err := h.postService.GetPostBySlug(c.Context(), slug, true) // true = increment views
	if err != nil {
		return response.Error(c, err)
	}

	// Only return published posts to public
	if post.Status != "published" {
		return response.Error(c, fiber.NewError(fiber.StatusNotFound, "Post not found"))
	}

	return response.SuccessData(c, post)
}

func (h *BlogHandler) SearchPosts(c *fiber.Ctx) error {
	query := c.Query("q")
	if query == "" {
		return response.Error(c, fiber.NewError(fiber.StatusBadRequest, "Search query required"))
	}

	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	posts, err := h.postService.SearchPosts(c.Context(), query, limit)
	if err != nil {
		return response.Error(c, err)
	}

	// Filter only published
	published := make([]*domain.BlogPost, 0)
	for _, post := range posts {
		if post.Status == "published" {
			published = append(published, post)
			if len(published) >= limit {
				break
			}
		}
	}

	return response.SuccessData(c, fiber.Map{
		"results": published,
		"query":   query,
		"total":   len(published),
	})
}

// ── Categories ────────────────────────────────────────────────────────────────

func (h *BlogHandler) GetAllCategories(c *fiber.Ctx) error {
	categories, _, err := h.categoryService.GetAllCategories(
		c.Context(),
		true, // activeOnly = true for public
		1000,
		0,
	)
	if err != nil {
		return response.Error(c, err)
	}

	return response.SuccessData(c, categories)
}

func (h *BlogHandler) GetCategoryBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")

	category, err := h.categoryService.GetCategoryBySlug(c.Context(), slug)
	if err != nil {
		return response.Error(c, err)
	}

	if !category.IsActive {
		return response.Error(c, fiber.NewError(fiber.StatusNotFound, "Category not found"))
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

	posts, _, err := h.postService.GetPostsByCategory( // ✅ Use _ to ignore total
		c.Context(),
		slug,
		params.Limit,
		params.GetOffset(),
	)
	if err != nil {
		return response.Error(c, err)
	}

	// Filter only published posts
	published := make([]*domain.BlogPost, 0)
	for _, post := range posts {
		if post.Status == "published" {
			published = append(published, post)
		}
	}

	// ✅ Use len(published) as the new total
	return response.Paginated(c, published, len(published), params.Page, params.Limit)
}

// ── Authors ───────────────────────────────────────────────────────────────────

func (h *BlogHandler) GetAllAuthors(c *fiber.Ctx) error {
	authors, _, err := h.authorService.GetAllAuthors(
		c.Context(),
		true, // activeOnly = true for public
		1000,
		0,
	)
	if err != nil {
		return response.Error(c, err)
	}

	return response.SuccessData(c, authors)
}

func (h *BlogHandler) GetAuthorBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")

	author, err := h.authorService.GetAuthorBySlug(c.Context(), slug)
	if err != nil {
		return response.Error(c, err)
	}

	if !author.IsActive {
		return response.Error(c, fiber.NewError(fiber.StatusNotFound, "Author not found"))
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

	posts, _, err := h.postService.GetPostsByAuthor( // ✅ Use _ to ignore total
		c.Context(),
		slug,
		params.Limit,
		params.GetOffset(),
	)
	if err != nil {
		return response.Error(c, err)
	}

	// Filter only published posts
	published := make([]*domain.BlogPost, 0)
	for _, post := range posts {
		if post.Status == "published" {
			published = append(published, post)
		}
	}

	// ✅ Use len(published) as the new total
	return response.Paginated(c, published, len(published), params.Page, params.Limit)
}

// ── Tags ──────────────────────────────────────────────────────────────────────

func (h *BlogHandler) GetPostsByTag(c *fiber.Ctx) error {
	tag := c.Params("tag")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "12"))

	params := &domain.PaginationParams{
		Page:  page,
		Limit: limit,
	}
	params.Validate()

	posts, _, err := h.postService.GetPostsByTag( // ✅ Use _ to ignore total
		c.Context(),
		tag,
		params.Limit,
		params.GetOffset(),
	)
	if err != nil {
		return response.Error(c, err)
	}

	// Filter only published posts
	published := make([]*domain.BlogPost, 0)
	for _, post := range posts {
		if post.Status == "published" {
			published = append(published, post)
		}
	}

	// ✅ Use len(published) as the new total
	return response.Paginated(c, published, len(published), params.Page, params.Limit)
}