package admin

import (
	"strconv"

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

type TemplateHandler struct {
	templateService *service.TemplateService
}

func NewTemplateHandler(templateService *service.TemplateService) *TemplateHandler {
	return &TemplateHandler{templateService: templateService}
}

func (h *TemplateHandler) GetAll(c *fiber.Ctx) error {
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
	if sort := c.Query("sort"); sort != "" {
		filters["sort"] = sort
	}

	templates, total, err := h.templateService.GetAllTemplates(c.Context(), filters, params.Limit, params.GetOffset())
	if err != nil {
		// ADD DETAILED LOGGING
		logger.Error("Failed to get templates",
			zap.Error(err),
			zap.Any("filters", filters),
			zap.Int("limit", params.Limit),
			zap.Int("offset", params.GetOffset()),
		)
		return response.Error(c, err)
	}

	return response.Paginated(c, templates, total, params.Page, params.Limit)
}

func (h *TemplateHandler) GetByID(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, fiber.NewError(400, "Invalid template ID"))
	}

	template, err := h.templateService.GetTemplateByID(c.Context(), int64(id))
	if err != nil {
		return response.Error(c, err)
	}

	return response.SuccessData(c, template)
}

type CreateTemplateRequest struct {
	Slug                string   `json:"slug" validate:"slug"`
	Title               string   `json:"title" validate:"required"`
	Description         string   `json:"description"`
	DetailedDescription string   `json:"detailed_description"`
	PriceINR            int      `json:"price_inr" validate:"required,min=0"`
	CategoryID          int64    `json:"category_id" validate:"required"`
	Tags                []string `json:"tags"`
	Status              string   `json:"status" validate:"required,oneof=draft active inactive"`
	IsFeatured          bool     `json:"is_featured"`
	MetaTitle           string   `json:"meta_title"`
	MetaDescription     string   `json:"meta_description"`
}

func (h *TemplateHandler) Create(c *fiber.Ctx) error {
	var req CreateTemplateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, err)
	}

	if err := validator.Validate(req); err != nil {
		return response.ValidationError(c, validator.FormatValidationErrors(err))
	}

	// TODO: Handle file uploads (thumbnail, preview images, template file)

	template := &domain.Template{
		Slug:                req.Slug,
		Title:               req.Title,
		Description:         &req.Description,
		DetailedDescription: &req.DetailedDescription,
		PriceINR:            req.PriceINR,
		CategoryID:          req.CategoryID,
		Tags:                pq.StringArray(req.Tags),
		Status:              req.Status,
		IsFeatured:          req.IsFeatured,
		MetaTitle:           &req.MetaTitle,
		MetaDescription:     &req.MetaDescription,
		FileURL:             "https://placeholder.com/file.pdf",
		PreviewURLs:         pq.StringArray([]string{}),
	}

	adminID := middleware.GetAdminID(c)

	if err := h.templateService.CreateTemplate(c.Context(), template, adminID); err != nil {
		return response.Error(c, err)
	}

	return response.Created(c, "Template created successfully", template)
}

func (h *TemplateHandler) Update(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, fiber.NewError(400, "Invalid template ID"))
	}

	var req CreateTemplateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, err)
	}

	if err := validator.Validate(req); err != nil {
		return response.ValidationError(c, validator.FormatValidationErrors(err))
	}

	template := &domain.Template{
		ID:                  int64(id),
		Slug:                req.Slug,
		Title:               req.Title,
		Description:         &req.Description,
		DetailedDescription: &req.DetailedDescription,
		PriceINR:            req.PriceINR,
		CategoryID:          req.CategoryID,
		Tags:                pq.StringArray(req.Tags),
		Status:              req.Status,
		IsFeatured:          req.IsFeatured,
		MetaTitle:           &req.MetaTitle,
		MetaDescription:     &req.MetaDescription,
		FileURL:             "https://placeholder.com/file.pdf",
	}

	adminID := middleware.GetAdminID(c)

	if err := h.templateService.UpdateTemplate(c.Context(), template, adminID); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, "Template updated successfully", template)
}

func (h *TemplateHandler) Delete(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, fiber.NewError(400, "Invalid template ID"))
	}

	adminID := middleware.GetAdminID(c)

	if err := h.templateService.DeleteTemplate(c.Context(), int64(id), adminID); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, "Template deleted successfully", nil)
}

func (h *TemplateHandler) GetAnalytics(c *fiber.Ctx) error {
	logger.Info("Attempting to get template analytics")
	
	analytics, err := h.templateService.GetAnalytics(c.Context())
	if err != nil {
		logger.Error("Failed to get template analytics", 
			zap.Error(err),
			zap.String("error_message", err.Error()),
		)
		
		// In development, return actual error
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),  // Show actual error
			"code":    "ANALYTICS_ERROR",
		})
	}

	logger.Info("Template analytics retrieved successfully", 
		zap.Any("analytics", analytics),
	)
	return response.SuccessData(c, analytics)
}
