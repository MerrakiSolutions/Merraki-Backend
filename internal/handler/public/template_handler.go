package public

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/merraki/merraki-backend/internal/domain"
	"github.com/merraki/merraki-backend/internal/pkg/response"
	"github.com/merraki/merraki-backend/internal/service"
)

type TemplateHandler struct {
	templateService *service.TemplateService
	categoryService *service.CategoryService
}

func NewTemplateHandler(
	templateService *service.TemplateService,
	categoryService *service.CategoryService,
) *TemplateHandler {
	return &TemplateHandler{
		templateService: templateService,
		categoryService: categoryService,
	}
}

func (h *TemplateHandler) GetAll(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	params := &domain.PaginationParams{
		Page:  page,
		Limit: limit,
	}
	params.Validate()

	filters := map[string]interface{}{
		"status": "active",
	}

	if categorySlug := c.Query("category"); categorySlug != "" {
		category, err := h.categoryService.GetTemplateCategoryBySlug(c.Context(), categorySlug)
		if err == nil && category != nil {
			// Check the actual type returned by GetTemplateCategoryBySlug
			if catMap, ok := category.(map[string]interface{}); ok {
				if id, exists := catMap["id"]; exists {
					filters["category_id"] = id
				}
			}
		}
	}

	if search := c.Query("search"); search != "" {
		filters["search"] = search
	}

	if minPrice := c.Query("min_price"); minPrice != "" {
		if price, err := strconv.Atoi(minPrice); err == nil {
			filters["min_price"] = price
		}
	}

	if maxPrice := c.Query("max_price"); maxPrice != "" {
		if price, err := strconv.Atoi(maxPrice); err == nil {
			filters["max_price"] = price
		}
	}

	if sort := c.Query("sort"); sort != "" {
		filters["sort"] = sort
	}

	if featured := c.Query("featured"); featured == "true" {
		filters["featured"] = true
	}

	templates, total, err := h.templateService.GetAllTemplates(c.Context(), filters, params.Limit, params.GetOffset())
	if err != nil {
		return response.Error(c, err)
	}

	return response.Paginated(c, templates, total, params.Page, params.Limit)
}

func (h *TemplateHandler) GetBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")

	template, err := h.templateService.GetTemplateBySlug(c.Context(), slug, true)
	if err != nil {
		return response.Error(c, err)
	}

	return response.SuccessData(c, template)
}

func (h *TemplateHandler) GetFeatured(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "6"))

	templates, err := h.templateService.GetFeaturedTemplates(c.Context(), limit)
	if err != nil {
		return response.Error(c, err)
	}

	return response.SuccessData(c, templates)
}

func (h *TemplateHandler) GetPopular(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	templates, err := h.templateService.GetPopularTemplates(c.Context(), limit)
	if err != nil {
		return response.Error(c, err)
	}

	return response.SuccessData(c, templates)
}

func (h *TemplateHandler) Search(c *fiber.Ctx) error {
	query := c.Query("q")
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	templates, err := h.templateService.SearchTemplates(c.Context(), query, limit)
	if err != nil {
		return response.Error(c, err)
	}

	return response.SuccessData(c, fiber.Map{
		"results": templates,
		"total":   len(templates),
		"query":   query,
	})
}

func (h *TemplateHandler) GetCategories(c *fiber.Ctx) error {
	categories, err := h.categoryService.GetTemplateCategories(c.Context(), true)
	if err != nil {
		return response.Error(c, err)
	}

	return response.SuccessData(c, categories)
}
