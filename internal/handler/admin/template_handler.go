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
// ADMIN TEMPLATE HANDLER - Template management
// ============================================================================

type TemplateHandler struct {
	templateService *service.TemplateService
	storageService  *service.StorageService
}

func NewTemplateHandler(
	templateService *service.TemplateService,
	storageService *service.StorageService,
) *TemplateHandler {
	return &TemplateHandler{
		templateService: templateService,
		storageService:  storageService,
	}
}

// stringValue returns the value of a pointer to a string, or an empty string if nil
func stringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// ============================================================================
// GET ALL TEMPLATES (ADMIN)
// ============================================================================

// GET /api/v1/admin/templates?status=active&page=1&limit=20
func (h *TemplateHandler) GetAllTemplates(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// Build filters
	filters := make(map[string]interface{})

	if status := c.Query("status"); status != "" {
		filters["status"] = domain.TemplateStatus(status)
	}

	if categoryIDStr := c.Query("category_id"); categoryIDStr != "" {
		if categoryID, err := strconv.ParseInt(categoryIDStr, 10, 64); err == nil {
			filters["category_id"] = categoryID
		}
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
// GET TEMPLATE BY ID
// ============================================================================

// GET /api/v1/admin/templates/:id
func (h *TemplateHandler) GetTemplateByID(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid template ID",
		})
	}

	template, err := h.templateService.GetTemplateByID(c.Context(), id, false)
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
// CREATE TEMPLATE
// ============================================================================

type CreateTemplateRequest struct {
	Name             string                `json:"name" validate:"required"`
	Slug             string                `json:"slug"`
	Tagline          *string               `json:"tagline"`
	Description      *string               `json:"description"`
	CategoryID       *int64                `json:"category_id"`
	PriceINR         float64               `json:"price_inr" validate:"required,min=0"`
	PriceUSD         float64               `json:"price_usd" validate:"required,min=0"`
	SalePriceINR     *float64              `json:"sale_price_inr"`
	SalePriceUSD     *float64              `json:"sale_price_usd"`
	IsOnSale         bool                  `json:"is_on_sale"`
	FileURL          *string               `json:"file_url"`
	FileSizeMB       *float64              `json:"file_size_mb"`
	FileFormat       *string               `json:"file_format"`
	PreviewURL       *string               `json:"preview_url"`
	StockQuantity    int                   `json:"stock_quantity"`
	IsUnlimitedStock bool                  `json:"is_unlimited_stock"`
	Status           domain.TemplateStatus `json:"status"`
	IsAvailable      bool                  `json:"is_available"`
	IsFeatured       bool                  `json:"is_featured"`
	IsBestseller     bool                  `json:"is_bestseller"`
	IsNew            bool                  `json:"is_new"`
	MetaTitle        *string               `json:"meta_title"`
	MetaDescription  *string               `json:"meta_description"`
	CurrentVersion   string                `json:"current_version"`
}

// POST /api/v1/admin/templates
func (h *TemplateHandler) CreateTemplate(c *fiber.Ctx) error {
	var req CreateTemplateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Get admin ID from context
	adminID := c.Locals("admin_id").(int64)

	// Build domain template
	template := &domain.Template{
		Name:             req.Name,
		Slug:             req.Slug,
		Tagline:          req.Tagline,
		Description:      stringValue(req.Description),
		CategoryID:       req.CategoryID,
		PriceINR:         req.PriceINR,
		PriceUSD:         req.PriceUSD,
		SalePriceINR:     req.SalePriceINR,
		SalePriceUSD:     req.SalePriceUSD,
		IsOnSale:         req.IsOnSale,
		FileURL:          req.FileURL,
		FileSizeMB:       req.FileSizeMB,
		FileFormat:       req.FileFormat,
		PreviewURL:       req.PreviewURL,
		StockQuantity:    req.StockQuantity,
		IsUnlimitedStock: req.IsUnlimitedStock,
		Status:           req.Status,
		IsAvailable:      req.IsAvailable,
		IsFeatured:       req.IsFeatured,
		IsBestseller:     req.IsBestseller,
		IsNew:            req.IsNew,
		MetaTitle:        req.MetaTitle,
		MetaDescription:  req.MetaDescription,
		CurrentVersion:   req.CurrentVersion,
	}

	err := h.templateService.CreateTemplate(c.Context(), template, adminID)
	if err != nil {
		logger.Error("Failed to create template", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create template",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"template": template,
	})
}

// ============================================================================
// UPDATE TEMPLATE
// ============================================================================

// PUT /api/v1/admin/templates/:id
func (h *TemplateHandler) UpdateTemplate(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid template ID",
		})
	}

	var req CreateTemplateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	adminID := c.Locals("admin_id").(int64)

	template := &domain.Template{
		ID:               id,
		Name:             req.Name,
		Slug:             req.Slug,
		Tagline:          req.Tagline,
		Description:      stringValue(req.Description),
		CategoryID:       req.CategoryID,
		PriceINR:         req.PriceINR,
		PriceUSD:         req.PriceUSD,
		SalePriceINR:     req.SalePriceINR,
		SalePriceUSD:     req.SalePriceUSD,
		IsOnSale:         req.IsOnSale,
		FileURL:          req.FileURL,
		FileSizeMB:       req.FileSizeMB,
		FileFormat:       req.FileFormat,
		PreviewURL:       req.PreviewURL,
		StockQuantity:    req.StockQuantity,
		IsUnlimitedStock: req.IsUnlimitedStock,
		Status:           req.Status,
		IsAvailable:      req.IsAvailable,
		IsFeatured:       req.IsFeatured,
		IsBestseller:     req.IsBestseller,
		IsNew:            req.IsNew,
		MetaTitle:        req.MetaTitle,
		MetaDescription:  req.MetaDescription,
		CurrentVersion:   req.CurrentVersion,
	}

	err = h.templateService.UpdateTemplate(c.Context(), template, adminID)
	if err != nil {
		logger.Error("Failed to update template", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update template",
		})
	}

	return c.JSON(fiber.Map{
		"template": template,
	})
}

// ============================================================================
// PATCH TEMPLATE (Partial update)
// ============================================================================

// PATCH /api/v1/admin/templates/:id
func (h *TemplateHandler) PatchTemplate(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid template ID",
		})
	}

	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	adminID := c.Locals("admin_id").(int64)

	err = h.templateService.PatchTemplate(c.Context(), id, updates, adminID)
	if err != nil {
		logger.Error("Failed to patch template", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update template",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Template updated successfully",
	})
}

// ============================================================================
// DELETE TEMPLATE
// ============================================================================

// DELETE /api/v1/admin/templates/:id
func (h *TemplateHandler) DeleteTemplate(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid template ID",
		})
	}

	adminID := c.Locals("admin_id").(int64)

	err = h.templateService.DeleteTemplate(c.Context(), id, adminID)
	if err != nil {
		logger.Error("Failed to delete template", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete template",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Template deleted successfully",
	})
}

// ============================================================================
// UPLOAD TEMPLATE FILE
// ============================================================================

// POST /api/v1/admin/templates/:id/upload-file
func (h *TemplateHandler) UploadTemplateFile(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid template ID",
		})
	}

	// Get file from request
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "File is required",
		})
	}

	// Upload to storage
	result, err := h.storageService.UploadFile(c.Context(), file, "templates")
	if err != nil {
		logger.Error("Failed to upload file", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to upload file",
		})
	}

	// Update template
	adminID := c.Locals("admin_id").(int64)
	fileSizeMB := float64(result.Bytes) / (1024 * 1024)

	updates := map[string]interface{}{
		"file_url":     result.PublicID,
		"file_size_mb": fileSizeMB,
		"file_format":  result.Format,
	}

	err = h.templateService.PatchTemplate(c.Context(), id, updates, adminID)
	if err != nil {
		logger.Error("Failed to update template", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update template",
		})
	}

	return c.JSON(fiber.Map{
		"success":   true,
		"file_url":  result.URL,
		"public_id": result.PublicID,
		"size_mb":   fileSizeMB,
	})
}

// ============================================================================
// ADD IMAGE
// ============================================================================

type AddImageRequest struct {
	URL          string  `json:"url" validate:"required"`
	AltText      *string `json:"alt_text"`
	DisplayOrder int     `json:"display_order"`
	IsPrimary    bool    `json:"is_primary"`
}

// POST /api/v1/admin/templates/:id/images
func (h *TemplateHandler) AddImage(c *fiber.Ctx) error {
	idStr := c.Params("id")
	templateID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid template ID",
		})
	}

	var req AddImageRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	adminID := c.Locals("admin_id").(int64)

	image := &domain.TemplateImage{
		TemplateID:   templateID,
		URL:          req.URL,
		AltText:      req.AltText,
		DisplayOrder: req.DisplayOrder,
		IsPrimary:    req.IsPrimary,
	}

	err = h.templateService.AddImage(c.Context(), image, adminID)
	if err != nil {
		logger.Error("Failed to add image", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to add image",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"image": image,
	})
}

// ============================================================================
// DELETE IMAGE
// ============================================================================

// DELETE /api/v1/admin/templates/images/:id
func (h *TemplateHandler) DeleteImage(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid image ID",
		})
	}

	adminID := c.Locals("admin_id").(int64)

	err = h.templateService.DeleteImage(c.Context(), id, adminID)
	if err != nil {
		logger.Error("Failed to delete image", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete image",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Image deleted successfully",
	})
}

// ============================================================================
// ADD FEATURE
// ============================================================================

type AddFeatureRequest struct {
	Title        string  `json:"title" validate:"required"`
	Description  *string `json:"description"`
	DisplayOrder int     `json:"display_order"`
}

// POST /api/v1/admin/templates/:id/features
func (h *TemplateHandler) AddFeature(c *fiber.Ctx) error {
	idStr := c.Params("id")
	templateID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid template ID",
		})
	}

	var req AddFeatureRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	adminID := c.Locals("admin_id").(int64)

	feature := &domain.TemplateFeature{
		TemplateID:   templateID,
		Title:        req.Title,
		Description:  req.Description,
		DisplayOrder: req.DisplayOrder,
	}

	err = h.templateService.AddFeature(c.Context(), feature, adminID)
	if err != nil {
		logger.Error("Failed to add feature", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to add feature",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"feature": feature,
	})
}

// ============================================================================
// DELETE FEATURE
// ============================================================================

// DELETE /api/v1/admin/templates/features/:id
func (h *TemplateHandler) DeleteFeature(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid feature ID",
		})
	}

	adminID := c.Locals("admin_id").(int64)

	err = h.templateService.DeleteFeature(c.Context(), id, adminID)
	if err != nil {
		logger.Error("Failed to delete feature", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete feature",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Feature deleted successfully",
	})
}

// ============================================================================
// UPDATE TAGS
// ============================================================================

type UpdateTagsRequest struct {
	Tags []string `json:"tags" validate:"required"`
}

// PUT /api/v1/admin/templates/:id/tags
func (h *TemplateHandler) UpdateTags(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid template ID",
		})
	}

	var req UpdateTagsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	adminID := c.Locals("admin_id").(int64)

	err = h.templateService.UpdateTags(c.Context(), id, req.Tags, adminID)
	if err != nil {
		logger.Error("Failed to update tags", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update tags",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Tags updated successfully",
	})
}
