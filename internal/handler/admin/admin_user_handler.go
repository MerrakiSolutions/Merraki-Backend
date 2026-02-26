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

type AdminUserHandler struct {
	adminService *service.AdminService
}

func NewAdminUserHandler(adminService *service.AdminService) *AdminUserHandler {
	return &AdminUserHandler{adminService: adminService}
}

func (h *AdminUserHandler) GetAll(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	params := &domain.PaginationParams{
		Page:  page,
		Limit: limit,
	}
	params.Validate()

	filters := make(map[string]interface{})
	if role := c.Query("role"); role != "" {
		filters["role"] = role
	}
	if search := c.Query("search"); search != "" {
		filters["search"] = search
	}

	admins, total, err := h.adminService.GetAllAdmins(c.Context(), filters, params.Limit, params.GetOffset())
	if err != nil {
		return response.Error(c, err)
	}

	return response.Paginated(c, admins, total, params.Page, params.Limit)
}

func (h *AdminUserHandler) GetByID(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, fiber.NewError(400, "Invalid admin ID"))
	}

	admin, err := h.adminService.GetAdminByID(c.Context(), int64(id))
	if err != nil {
		return response.Error(c, err)
	}

	return response.SuccessData(c, admin)
}

type CreateAdminRequest struct {
	Email       string                 `json:"email" validate:"required,email"`
	Name        string                 `json:"name" validate:"required"`
	Password    string                 `json:"password" validate:"required,password"`
	Role        string                 `json:"role" validate:"required,oneof=super_admin admin content_manager order_manager support_staff"`
	Permissions map[string]interface{} `json:"permissions"`
	IsActive    bool                   `json:"is_active"`
}

func (h *AdminUserHandler) Create(c *fiber.Ctx) error {
	var req CreateAdminRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, err)
	}

	if err := validator.Validate(req); err != nil {
		return response.ValidationError(c, validator.FormatValidationErrors(err))
	}

	createdBy := middleware.GetAdminID(c)

	createReq := &service.CreateAdminRequest{
		Email:       req.Email,
		Name:        req.Name,
		Password:    req.Password,
		Role:        req.Role,
		Permissions: req.Permissions,
		IsActive:    req.IsActive,
		CreatedBy:   createdBy,
	}

	admin, err := h.adminService.CreateAdmin(c.Context(), createReq)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Created(c, "Admin created successfully", admin)
}

type UpdateAdminRequest struct {
	Name        string                 `json:"name" validate:"required"`
	Role        string                 `json:"role" validate:"required"`
	Permissions map[string]interface{} `json:"permissions"`
	IsActive    bool                   `json:"is_active"`
}

func (h *AdminUserHandler) Update(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, fiber.NewError(400, "Invalid admin ID"))
	}

	var req UpdateAdminRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, err)
	}

	if err := validator.Validate(req); err != nil {
		return response.ValidationError(c, validator.FormatValidationErrors(err))
	}

	updatedBy := middleware.GetAdminID(c)

	admin := &domain.Admin{
		ID:          int64(id),
		Name:        req.Name,
		Role:        req.Role,
		Permissions: req.Permissions,
		IsActive:    req.IsActive,
	}

	if err := h.adminService.UpdateAdmin(c.Context(), admin, updatedBy); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, "Admin updated successfully", admin)
}

func (h *AdminUserHandler) Delete(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, fiber.NewError(400, "Invalid admin ID"))
	}

	deletedBy := middleware.GetAdminID(c)

	// Cannot delete self
	if int64(id) == deletedBy {
		return response.Error(c, fiber.NewError(400, "Cannot delete your own account"))
	}

	if err := h.adminService.DeleteAdmin(c.Context(), int64(id), deletedBy); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, "Admin deleted successfully", nil)
}