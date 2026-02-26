package public

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/merraki/merraki-backend/internal/domain"
	"github.com/merraki/merraki-backend/internal/pkg/response"
	"github.com/merraki/merraki-backend/internal/service"
)

type BlogHandler struct {
	blogService     *service.BlogService
	categoryService *service.CategoryService
}

func NewBlogHandler(
	blogService *service.BlogService,
	categoryService *service.CategoryService,
) *BlogHandler {
	return &BlogHandler{
		blogService:     blogService,
		categoryService: categoryService,
	}
}

func (h *BlogHandler) GetAll(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	params := &domain.PaginationParams{
		Page:  page,
		Limit: limit,
	}
	params.Validate()

	filters := map[string]interface{}{
		"status": "published",
	}

	if category := c.Query("category"); category != "" {
		// TODO: Get category by slug
	}

	if tag := c.Query("tag"); tag != "" {
		filters["tag"] = tag
	}

	if search := c.Query("search"); search != "" {
		filters["search"] = search
	}

	if sort := c.Query("sort"); sort != "" {
		filters["sort"] = sort
	}

	posts, total, err := h.blogService.GetAllPosts(c.Context(), filters, params.Limit, params.GetOffset())
	if err != nil {
		return response.Error(c, err)
	}

	return response.Paginated(c, posts, total, params.Page, params.Limit)
}

func (h *BlogHandler) GetBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")

	post, err := h.blogService.GetPostBySlug(c.Context(), slug, true)
	if err != nil {
		return response.Error(c, err)
	}

	return response.SuccessData(c, post)
}

func (h *BlogHandler) GetFeatured(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "3"))

	posts, err := h.blogService.GetFeaturedPosts(c.Context(), limit)
	if err != nil {
		return response.Error(c, err)
	}

	return response.SuccessData(c, posts)
}

func (h *BlogHandler) GetPopular(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "5"))

	posts, err := h.blogService.GetPopularPosts(c.Context(), limit)
	if err != nil {
		return response.Error(c, err)
	}

	return response.SuccessData(c, posts)
}

func (h *BlogHandler) Search(c *fiber.Ctx) error {
	query := c.Query("q")
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	posts, err := h.blogService.SearchPosts(c.Context(), query, limit)
	if err != nil {
		return response.Error(c, err)
	}

	return response.SuccessData(c, fiber.Map{
		"results": posts,
		"total":   len(posts),
		"query":   query,
	})
}

func (h *BlogHandler) GetCategories(c *fiber.Ctx) error {
	categories, err := h.categoryService.GetBlogCategories(c.Context(), true)
	if err != nil {
		return response.Error(c, err)
	}

	return response.SuccessData(c, categories)
}