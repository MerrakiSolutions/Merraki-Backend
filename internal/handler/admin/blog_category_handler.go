package admin

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/merraki/merraki-backend/internal/domain"
	"github.com/merraki/merraki-backend/internal/middleware"
	"github.com/merraki/merraki-backend/internal/pkg/response"
	"github.com/merraki/merraki-backend/internal/pkg/validator"
	"github.com/merraki/merraki-backend/internal/service"
)

type BlogCategoryHandler struct {
	categoryService *service.BlogCategoryService
}

func NewBlogCategoryHandler(categoryService *service.BlogCategoryService) *BlogCategoryHandler {
	return &BlogCategoryHandler{categoryService: categoryService}
}

type CreateBlogCategoryRequest struct {
	Name         string  `json:"name" validate:"required"`
	Slug         string  `json:"slug"`
	Description  string  `json:"description"`
	ParentID     *int64  `json:"parent_id"`
	DisplayOrder int     `json:"display_order"`
	IsActive     bool    `json:"is_active"`
}

func (h *BlogCategoryHandler) GetAll(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	activeOnly := c.Query("active_only") == "true"

	params := &domain.PaginationParams{
		Page:  page,
		Limit: limit,
	}
	params.Validate()

	categories, total, err := h.categoryService.GetAllCategories(c.Context(), activeOnly, params.Limit, params.GetOffset())
	if err != nil {
		return response.Error(c, err)
	}

	return response.Paginated(c, categories, total, params.Page, params.Limit)
}

func (h *BlogCategoryHandler) GetByID(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, fiber.NewError(400, "Invalid category ID"))
	}

	category, err := h.categoryService.GetCategoryByID(c.Context(), int64(id))
	if err != nil {
		return response.Error(c, err)
	}

	return response.SuccessData(c, category)
}

func (h *BlogCategoryHandler) GetBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")

	category, err := h.categoryService.GetCategoryBySlug(c.Context(), slug)
	if err != nil {
		return response.Error(c, err)
	}

	return response.SuccessData(c, category)
}

func (h *BlogCategoryHandler) Create(c *fiber.Ctx) error {
	var req CreateBlogCategoryRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, err)
	}

	if err := validator.Validate(req); err != nil {
		return response.ValidationError(c, validator.FormatValidationErrors(err))
	}

	adminID := middleware.GetAdminID(c)

	category := &domain.BlogCategory{
		Name:         req.Name,
		Slug:         req.Slug,
		Description:  &req.Description,
		ParentID:     req.ParentID,
		DisplayOrder: req.DisplayOrder,
		IsActive:     req.IsActive,
	}

	if err := h.categoryService.CreateCategory(c.Context(), category, adminID); err != nil {
		return response.Error(c, err)
	}

	return response.Created(c, "Category created successfully", category)
}

func (h *BlogCategoryHandler) Update(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, fiber.NewError(400, "Invalid category ID"))
	}

	var req CreateBlogCategoryRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, err)
	}

	if err := validator.Validate(req); err != nil {
		return response.ValidationError(c, validator.FormatValidationErrors(err))
	}

	adminID := middleware.GetAdminID(c)

	category := &domain.BlogCategory{
		ID:           int64(id),
		Name:         req.Name,
		Slug:         req.Slug,
		Description:  &req.Description,
		ParentID:     req.ParentID,
		DisplayOrder: req.DisplayOrder,
		IsActive:     req.IsActive,
	}

	if err := h.categoryService.UpdateCategory(c.Context(), category, adminID); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, "Category updated successfully", category)
}

func (h *BlogCategoryHandler) Delete(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, fiber.NewError(400, "Invalid category ID"))
	}

	adminID := middleware.GetAdminID(c)

	if err := h.categoryService.DeleteCategory(c.Context(), int64(id), adminID); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, "Category deleted successfully", nil)
}