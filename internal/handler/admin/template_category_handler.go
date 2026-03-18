package admin

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/merraki/merraki-backend/internal/domain"
	"github.com/merraki/merraki-backend/internal/pkg/logger"
	"github.com/merraki/merraki-backend/internal/service"
	"go.uber.org/zap"
)

// ============================================================================
// ADMIN CATEGORY HANDLER - Category management
// ============================================================================

type CategoryHandler struct {
	categoryService *service.CategoryService
}

func NewCategoryHandler(categoryService *service.CategoryService) *CategoryHandler {
	return &CategoryHandler{
		categoryService: categoryService,
	}
}

// ============================================================================
// GET ALL CATEGORIES
// ============================================================================

// GET /api/v1/admin/categories
func (h *CategoryHandler) GetAllCategories(c *fiber.Ctx) error {
	activeOnlyStr := c.Query("active_only", "false")
	activeOnly := activeOnlyStr == "true"

	categories, err := h.categoryService.GetAllCategories(c.Context(), activeOnly)
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
// GET CATEGORY BY ID
// ============================================================================

// GET /api/v1/admin/categories/:id
func (h *CategoryHandler) GetCategoryByID(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid category ID",
		})
	}

	category, err := h.categoryService.GetCategoryByID(c.Context(), id)
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
// CREATE CATEGORY
// ============================================================================

type CreateCategoryRequest struct {
	Name            string  `json:"name" validate:"required"`
	Slug            string  `json:"slug"`
	Description     *string `json:"description"`
	ParentID        *int64  `json:"parent_id"`
	DisplayOrder    int     `json:"display_order"`
	IsActive        bool    `json:"is_active"`
	MetaTitle       *string `json:"meta_title"`
	MetaDescription *string `json:"meta_description"`
}

// POST /api/v1/admin/categories
func (h *CategoryHandler) CreateCategory(c *fiber.Ctx) error {
	var req CreateCategoryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	adminID := c.Locals("admin_id").(int64)

	category := &domain.Category{
		Name:            req.Name,
		Slug:            req.Slug,
		Description:     req.Description,
		ParentID:        req.ParentID,
		DisplayOrder:    req.DisplayOrder,
		IsActive:        req.IsActive,
		MetaTitle:       req.MetaTitle,
		MetaDescription: req.MetaDescription,
	}

	err := h.categoryService.CreateCategory(c.Context(), category, adminID)
	if err != nil {
		logger.Error("Failed to create category", zap.Error(err))
		
		if err == domain.ErrDuplicateEntry {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "Category with this slug already exists",
			})
		}
		
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create category",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"category": category,
	})
}

// ============================================================================
// UPDATE CATEGORY
// ============================================================================

// PUT /api/v1/admin/categories/:id
func (h *CategoryHandler) UpdateCategory(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid category ID",
		})
	}

	var req CreateCategoryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	adminID := c.Locals("admin_id").(int64)

	category := &domain.Category{
		ID:              id,
		Name:            req.Name,
		Slug:            req.Slug,
		Description:     req.Description,
		ParentID:        req.ParentID,
		DisplayOrder:    req.DisplayOrder,
		IsActive:        req.IsActive,
		MetaTitle:       req.MetaTitle,
		MetaDescription: req.MetaDescription,
	}

	err = h.categoryService.UpdateCategory(c.Context(), category, adminID)
	if err != nil {
		logger.Error("Failed to update category", zap.Error(err))
		
		if err == domain.ErrDuplicateEntry {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "Category with this slug already exists",
			})
		}
		
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update category",
		})
	}

	return c.JSON(fiber.Map{
		"category": category,
	})
}

// ============================================================================
// DELETE CATEGORY
// ============================================================================

// DELETE /api/v1/admin/categories/:id
func (h *CategoryHandler) DeleteCategory(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid category ID",
		})
	}

	adminID := c.Locals("admin_id").(int64)

	err = h.categoryService.DeleteCategory(c.Context(), id, adminID)
	if err != nil {
		logger.Error("Failed to delete category", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete category",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Category deleted successfully",
	})
}