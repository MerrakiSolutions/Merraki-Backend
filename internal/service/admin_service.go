package service

import (
	"context"

	"github.com/merraki/merraki-backend/internal/domain"
	"github.com/merraki/merraki-backend/internal/pkg/crypto"
	apperrors "github.com/merraki/merraki-backend/internal/pkg/errors"
	"github.com/merraki/merraki-backend/internal/repository/postgres"
)

type AdminService struct {
	adminRepo *postgres.AdminRepository
	logRepo   *postgres.ActivityLogRepository
}

func NewAdminService(
	adminRepo *postgres.AdminRepository,
	logRepo *postgres.ActivityLogRepository,
) *AdminService {
	return &AdminService{
		adminRepo: adminRepo,
		logRepo:   logRepo,
	}
}

type CreateAdminRequest struct {
	Email       string
	Name        string
	Password    string
	Role        string
	Permissions domain.JSONMap
	IsActive    bool
	CreatedBy   int64
}

func (s *AdminService) CreateAdmin(ctx context.Context, req *CreateAdminRequest) (*domain.Admin, error) {
	// Check if email exists
	existing, err := s.adminRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, apperrors.Wrap(err, "DATABASE_ERROR", "Failed to check email", 500)
	}

	if existing != nil {
		return nil, apperrors.New("EMAIL_EXISTS", "Email already registered", 409)
	}

	// Hash password
	passwordHash, err := crypto.HashPassword(req.Password, nil)
	if err != nil {
		return nil, apperrors.Wrap(err, "PASSWORD_ERROR", "Failed to hash password", 500)
	}

	// Create admin
	admin := &domain.Admin{
		Email:        req.Email,
		Name:         req.Name,
		PasswordHash: passwordHash,
		Role:         req.Role,
		Permissions:  req.Permissions,
		IsActive:     req.IsActive,
	}

	if err := s.adminRepo.Create(ctx, admin); err != nil {
		return nil, apperrors.Wrap(err, "DATABASE_ERROR", "Failed to create admin", 500)
	}

	// Log activity
	_ = s.logRepo.Create(ctx, &domain.ActivityLog{
		AdminID:    &req.CreatedBy,
		Action:     "create_admin",
		EntityType: strPtr("admin"),
		EntityID:   &admin.ID,
		Details:    domain.JSONMap{"new_admin_email": admin.Email, "role": admin.Role},
	})

	return admin, nil
}

func (s *AdminService) GetAdminByID(ctx context.Context, id int64) (*domain.Admin, error) {
	admin, err := s.adminRepo.FindByID(ctx, id)
	if err != nil {
		return nil, apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find admin", 500)
	}

	if admin == nil {
		return nil, apperrors.ErrNotFound
	}

	return admin, nil
}

func (s *AdminService) GetAllAdmins(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*domain.Admin, int, error) {
	return s.adminRepo.GetAll(ctx, filters, limit, offset)
}

func (s *AdminService) UpdateAdmin(ctx context.Context, admin *domain.Admin, updatedBy int64) error {
	// Check if admin exists
	existing, err := s.adminRepo.FindByID(ctx, admin.ID)
	if err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find admin", 500)
	}

	if existing == nil {
		return apperrors.ErrNotFound
	}

	// Update
	if err := s.adminRepo.Update(ctx, admin); err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to update admin", 500)
	}

	// Log activity
	_ = s.logRepo.Create(ctx, &domain.ActivityLog{
		AdminID:    &updatedBy,
		Action:     "update_admin",
		EntityType: strPtr("admin"),
		EntityID:   &admin.ID,
		Details:    domain.JSONMap{"updated_admin": admin.Email},
	})

	return nil
}

func (s *AdminService) DeleteAdmin(ctx context.Context, id, deletedBy int64) error {
	// Check if admin exists
	admin, err := s.adminRepo.FindByID(ctx, id)
	if err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find admin", 500)
	}

	if admin == nil {
		return apperrors.ErrNotFound
	}

	// Delete
	if err := s.adminRepo.Delete(ctx, id); err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to delete admin", 500)
	}

	// Log activity
	_ = s.logRepo.Create(ctx, &domain.ActivityLog{
		AdminID:    &deletedBy,
		Action:     "delete_admin",
		EntityType: strPtr("admin"),
		EntityID:   &id,
		Details:    domain.JSONMap{"deleted_admin": admin.Email},
	})

	return nil
}