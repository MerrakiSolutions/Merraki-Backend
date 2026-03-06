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

type BlogAuthorHandler struct {
	authorService *service.BlogAuthorService
}

func NewBlogAuthorHandler(authorService *service.BlogAuthorService) *BlogAuthorHandler {
	return &BlogAuthorHandler{authorService: authorService}
}

type CreateAuthorRequest struct {
	AdminID     *int64                 `json:"admin_id"`
	Name        string                 `json:"name" validate:"required"`
	Slug        string                 `json:"slug"`
	Email       string                 `json:"email" validate:"email"`
	Bio         string                 `json:"bio"`
	AvatarURL   string                 `json:"avatar_url"`
	SocialLinks map[string]interface{} `json:"social_links"`
	IsActive    bool                   `json:"is_active"`
}

func (h *BlogAuthorHandler) GetAll(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	activeOnly := c.Query("active_only") == "true"

	params := &domain.PaginationParams{
		Page:  page,
		Limit: limit,
	}
	params.Validate()

	authors, total, err := h.authorService.GetAllAuthors(c.Context(), activeOnly, params.Limit, params.GetOffset())
	if err != nil {
		return response.Error(c, err)
	}

	return response.Paginated(c, authors, total, params.Page, params.Limit)
}

func (h *BlogAuthorHandler) GetByID(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, fiber.NewError(400, "Invalid author ID"))
	}

	author, err := h.authorService.GetAuthorByID(c.Context(), int64(id))
	if err != nil {
		return response.Error(c, err)
	}

	return response.SuccessData(c, author)
}

func (h *BlogAuthorHandler) GetBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")

	author, err := h.authorService.GetAuthorBySlug(c.Context(), slug)
	if err != nil {
		return response.Error(c, err)
	}

	return response.SuccessData(c, author)
}

func (h *BlogAuthorHandler) Create(c *fiber.Ctx) error {
	var req CreateAuthorRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, err)
	}

	if err := validator.Validate(req); err != nil {
		return response.ValidationError(c, validator.FormatValidationErrors(err))
	}

	adminID := middleware.GetAdminID(c)

	author := &domain.BlogAuthor{
		AdminID:     req.AdminID,
		Name:        req.Name,
		Slug:        req.Slug,
		Email:       &req.Email,
		Bio:         &req.Bio,
		AvatarURL:   &req.AvatarURL,
		SocialLinks: req.SocialLinks,
		IsActive:    req.IsActive,
	}

	if err := h.authorService.CreateAuthor(c.Context(), author, adminID); err != nil {
		return response.Error(c, err)
	}

	return response.Created(c, "Author created successfully", author)
}

func (h *BlogAuthorHandler) Update(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, fiber.NewError(400, "Invalid author ID"))
	}

	var req CreateAuthorRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, err)
	}

	if err := validator.Validate(req); err != nil {
		return response.ValidationError(c, validator.FormatValidationErrors(err))
	}

	adminID := middleware.GetAdminID(c)

	author := &domain.BlogAuthor{
		ID:          int64(id),
		AdminID:     req.AdminID,
		Name:        req.Name,
		Slug:        req.Slug,
		Email:       &req.Email,
		Bio:         &req.Bio,
		AvatarURL:   &req.AvatarURL,
		SocialLinks: req.SocialLinks,
		IsActive:    req.IsActive,
	}

	if err := h.authorService.UpdateAuthor(c.Context(), author, adminID); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, "Author updated successfully", author)
}

func (h *BlogAuthorHandler) Delete(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, fiber.NewError(400, "Invalid author ID"))
	}

	adminID := middleware.GetAdminID(c)

	if err := h.authorService.DeleteAuthor(c.Context(), int64(id), adminID); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, "Author deleted successfully", nil)
}