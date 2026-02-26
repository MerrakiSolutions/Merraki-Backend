package admin

import (
	"github.com/gofiber/fiber/v2"
	"github.com/merraki/merraki-backend/internal/domain"
	"github.com/merraki/merraki-backend/internal/middleware"
	"github.com/merraki/merraki-backend/internal/pkg/response"
	"github.com/merraki/merraki-backend/internal/pkg/validator"
	"github.com/merraki/merraki-backend/internal/service"
)

type CategoryHandler struct {
	categoryService *service.CategoryService
}

func NewCategoryHandler(categoryService *service.CategoryService) *CategoryHandler {
	return &CategoryHandler{categoryService: categoryService}
}

// Template Categories
func (h *CategoryHandler) GetTemplateCategories(c *fiber.Ctx) error {
	categories, err := h.categoryService.GetTemplateCategories(c.Context(), false)
	if err != nil {
		return response.Error(c, err)
	}

	return response.SuccessData(c, categories)
}

type CreateTemplateCategoryRequest struct {
	Slug            string `json:"slug" validate:"slug"`
	Name            string `json:"name" validate:"required"`
	Description     string `json:"description"`
	IconName        string `json:"icon_name"`
	DisplayOrder    int    `json:"display_order"`
	ColorHex        string `json:"color_hex"`
	MetaTitle       string `json:"meta_title"`
	MetaDescription string `json:"meta_description"`
	IsActive        bool   `json:"is_active"`
}

func (h *CategoryHandler) CreateTemplateCategory(c *fiber.Ctx) error {
	var req CreateTemplateCategoryRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, err)
	}

	if err := validator.Validate(req); err != nil {
		return response.ValidationError(c, validator.FormatValidationErrors(err))
	}

	adminID := middleware.GetAdminID(c)

	category := &domain.TemplateCategory{
		Slug:            req.Slug,
		Name:            req.Name,
		Description:     &req.Description,
		IconName:        &req.IconName,
		DisplayOrder:    req.DisplayOrder,
		ColorHex:        &req.ColorHex,
		MetaTitle:       &req.MetaTitle,
		MetaDescription: &req.MetaDescription,
		IsActive:        req.IsActive,
	}

	if err := h.categoryService.CreateTemplateCategory(c.Context(), category, adminID); err != nil {
		return response.Error(c, err)
	}

	return response.Created(c, "Category created successfully", category)
}

func (h *CategoryHandler) UpdateTemplateCategory(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, fiber.NewError(400, "Invalid category ID"))
	}

	var req CreateTemplateCategoryRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, err)
	}

	if err := validator.Validate(req); err != nil {
		return response.ValidationError(c, validator.FormatValidationErrors(err))
	}

	adminID := middleware.GetAdminID(c)

	category := &domain.TemplateCategory{
		ID:              int64(id),
		Slug:            req.Slug,
		Name:            req.Name,
		Description:     &req.Description,
		IconName:        &req.IconName,
		DisplayOrder:    req.DisplayOrder,
		ColorHex:        &req.ColorHex,
		MetaTitle:       &req.MetaTitle,
		MetaDescription: &req.MetaDescription,
		IsActive:        req.IsActive,
	}

	if err := h.categoryService.UpdateTemplateCategory(c.Context(), category, adminID); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, "Category updated successfully", category)
}

func (h *CategoryHandler) DeleteTemplateCategory(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, fiber.NewError(400, "Invalid category ID"))
	}

	adminID := middleware.GetAdminID(c)

	if err := h.categoryService.DeleteTemplateCategory(c.Context(), int64(id), adminID); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, "Category deleted successfully", nil)
}

// Blog Categories
func (h *CategoryHandler) GetBlogCategories(c *fiber.Ctx) error {
	categories, err := h.categoryService.GetBlogCategories(c.Context(), false)
	if err != nil {
		return response.Error(c, err)
	}

	return response.SuccessData(c, categories)
}

type CreateBlogCategoryRequest struct {
	Slug         string `json:"slug" validate:"slug"`
	Name         string `json:"name" validate:"required"`
	Description  string `json:"description"`
	DisplayOrder int    `json:"display_order"`
	IsActive     bool   `json:"is_active"`
}

func (h *CategoryHandler) CreateBlogCategory(c *fiber.Ctx) error {
	var req CreateBlogCategoryRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, err)
	}

	if err := validator.Validate(req); err != nil {
		return response.ValidationError(c, validator.FormatValidationErrors(err))
	}

	adminID := middleware.GetAdminID(c)

	category := &domain.BlogCategory{
		Slug:         req.Slug,
		Name:         req.Name,
		Description:  &req.Description,
		DisplayOrder: req.DisplayOrder,
		IsActive:     req.IsActive,
	}

	if err := h.categoryService.CreateBlogCategory(c.Context(), category, adminID); err != nil {
		return response.Error(c, err)
	}

	return response.Created(c, "Blog category created successfully", category)
}

func (h *CategoryHandler) UpdateBlogCategory(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, fiber.NewError(400, "Invalid category ID"))
	}

	var req CreateBlogCategoryRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, err)
	}

	_ = middleware.GetAdminID(c)

	category := &domain.BlogCategory{
		ID:           int64(id),
		Name:         req.Name,
		Description:  &req.Description,
		DisplayOrder: req.DisplayOrder,
		IsActive:     req.IsActive,
	}

	if err := h.categoryService.UpdateBlogCategory(c.Context(), category); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, "Blog category updated successfully", category)
}

func (h *CategoryHandler) DeleteBlogCategory(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, fiber.NewError(400, "Invalid category ID"))
	}

	if err := h.categoryService.DeleteBlogCategory(c.Context(), int64(id)); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, "Blog category deleted successfully", nil)
}