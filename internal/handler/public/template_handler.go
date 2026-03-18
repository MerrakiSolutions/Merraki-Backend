package public

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/merraki/merraki-backend/internal/service"
)

// ============================================================================
// TEMPLATE HANDLER - Public template catalog
// ============================================================================

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

// ============================================================================
// GET ALL TEMPLATES
// ============================================================================

// GET /api/v1/templates?category_id=1&featured=true&search=budget&sort=price_asc&page=1&limit=12
func (h *TemplateHandler) GetAllTemplates(c *fiber.Ctx) error {
	// Parse query parameters
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "12"))
	
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 12
	}

	// Build filters
	filters := make(map[string]interface{})
	filters["available"] = true

	if categoryIDStr := c.Query("category_id"); categoryIDStr != "" {
		if categoryID, err := strconv.ParseInt(categoryIDStr, 10, 64); err == nil {
			filters["category_id"] = categoryID
		}
	}

	if featuredStr := c.Query("featured"); featuredStr == "true" {
		filters["featured"] = true
	}

	if search := c.Query("search"); search != "" {
		filters["search"] = search
	}

	if sort := c.Query("sort"); sort != "" {
		filters["sort"] = sort
	}

	offset := (page - 1) * limit

	templates, total, err := h.templateService.GetAllTemplatesWithRelations(
		c.Context(),
		filters,
		limit,
		offset,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get templates",
		})
	}

	return c.JSON(fiber.Map{
		"templates": templates,
		"total":     total,
		"page":      page,
		"limit":     limit,
	})
}

// ============================================================================
// GET TEMPLATE BY SLUG
// ============================================================================

// GET /api/v1/templates/:slug
func (h *TemplateHandler) GetTemplateBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")

	template, err := h.templateService.GetTemplateBySlug(c.Context(), slug, true)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Template not found",
		})
	}

	return c.JSON(fiber.Map{
		"template": template,
	})
}

// ============================================================================
// GET TEMPLATE BY ID
// ============================================================================

// GET /api/v1/templates/by-id/:id
func (h *TemplateHandler) GetTemplateByID(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid template ID",
		})
	}

	template, err := h.templateService.GetTemplateByID(c.Context(), id, true)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Template not found",
		})
	}

	return c.JSON(fiber.Map{
		"template": template,
	})
}

// ============================================================================
// GET FEATURED TEMPLATES
// ============================================================================

// GET /api/v1/templates/featured?limit=6
func (h *TemplateHandler) GetFeaturedTemplates(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "6"))

	if limit < 1 || limit > 50 {
		limit = 6
	}

	templates, err := h.templateService.GetFeaturedTemplates(c.Context(), limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get templates",
		})
	}

	return c.JSON(fiber.Map{
		"templates": templates,
	})
}

// ============================================================================
// GET BESTSELLERS
// ============================================================================

// GET /api/v1/templates/bestsellers?limit=6
func (h *TemplateHandler) GetBestsellers(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "6"))

	if limit < 1 || limit > 50 {
		limit = 6
	}

	templates, err := h.templateService.GetBestsellers(c.Context(), limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get bestsellers",
		})
	}

	return c.JSON(fiber.Map{
		"templates": templates,
	})
}

// ============================================================================
// GET NEW TEMPLATES
// ============================================================================

// GET /api/v1/templates/new?limit=6
func (h *TemplateHandler) GetNewTemplates(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "6"))

	if limit < 1 || limit > 50 {
		limit = 6
	}

	templates, err := h.templateService.GetNewTemplates(c.Context(), limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get new templates",
		})
	}

	return c.JSON(fiber.Map{
		"templates": templates,
	})
}

// ============================================================================
// GET CATEGORIES
// ============================================================================

// GET /api/v1/categories
func (h *TemplateHandler) GetCategories(c *fiber.Ctx) error {
	categories, err := h.categoryService.GetAllCategories(c.Context(), true)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get categories",
		})
	}

	return c.JSON(fiber.Map{
		"categories": categories,
	})
}

// ============================================================================
// GET CATEGORY BY SLUG
// ============================================================================

// GET /api/v1/categories/:slug
func (h *TemplateHandler) GetCategoryBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")

	category, err := h.categoryService.GetCategoryBySlug(c.Context(), slug)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Category not found",
		})
	}

	return c.JSON(fiber.Map{
		"category": category,
	})
}

// ============================================================================
// SEARCH TEMPLATES
// ============================================================================

// GET /api/v1/templates/search?q=budget&limit=10
func (h *TemplateHandler) SearchTemplates(c *fiber.Ctx) error {
	query := c.Query("q")
	if query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Search query is required",
		})
	}

	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	if limit < 1 || limit > 50 {
		limit = 10
	}

	templates, err := h.templateService.SearchTemplates(c.Context(), query, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Search failed",
		})
	}

	return c.JSON(fiber.Map{
		"templates": templates,
		"query":     query,
	})
}

// ============================================================================
// GET TEMPLATES BY CATEGORY
// ============================================================================

// GET /api/v1/templates/by-category/:slug?page=1&limit=12
func (h *TemplateHandler) GetTemplatesByCategory(c *fiber.Ctx) error {
	slug := c.Params("slug")
	
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "12"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 12
	}

	offset := (page - 1) * limit

	templates, total, err := h.templateService.GetTemplatesByCategory(
		c.Context(),
		slug,
		limit,
		offset,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get templates",
		})
	}

	return c.JSON(fiber.Map{
		"templates": templates,
		"total":     total,
		"page":      page,
		"limit":     limit,
	})
}

// ============================================================================
// GET TEMPLATES BY TAG
// ============================================================================

// GET /api/v1/templates/by-tag/:tag?page=1&limit=12
func (h *TemplateHandler) GetTemplatesByTag(c *fiber.Ctx) error {
	tag := c.Params("tag")
	
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "12"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 12
	}

	offset := (page - 1) * limit

	templates, total, err := h.templateService.GetTemplatesByTag(
		c.Context(),
		tag,
		limit,
		offset,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get templates",
		})
	}

	return c.JSON(fiber.Map{
		"templates": templates,
		"total":     total,
		"page":      page,
		"limit":     limit,
	})
}