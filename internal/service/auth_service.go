package service

import (
	"context"
	"time"

	"github.com/merraki/merraki-backend/internal/config"
	"github.com/merraki/merraki-backend/internal/domain"
	"github.com/merraki/merraki-backend/internal/pkg/crypto"
	apperrors "github.com/merraki/merraki-backend/internal/pkg/errors"
	"github.com/merraki/merraki-backend/internal/pkg/jwt"
	"github.com/merraki/merraki-backend/internal/repository/postgres"
)

type AuthService struct {
	adminRepo   *postgres.AdminRepository
	sessionRepo *postgres.SessionRepository
	logRepo     *postgres.ActivityLogRepository
	pasetoMaker *jwt.PasetoMaker
	cfg         *config.Config
}

func NewAuthService(
	adminRepo *postgres.AdminRepository,
	sessionRepo *postgres.SessionRepository,
	logRepo *postgres.ActivityLogRepository,
	cfg *config.Config,
) (*AuthService, error) {
	pasetoMaker, err := jwt.NewPasetoMaker(cfg.Auth.PasetoKey)
	if err != nil {
		return nil, err
	}

	return &AuthService{
		adminRepo:   adminRepo,
		sessionRepo: sessionRepo,
		logRepo:     logRepo,
		pasetoMaker: pasetoMaker,
		cfg:         cfg,
	}, nil
}

type LoginRequest struct {
	Email      string
	Password   string
	IPAddress  string
	UserAgent  string
	DeviceName string
}

type LoginResponse struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	ExpiresIn    int
	Admin        *domain.Admin
}

func (s *AuthService) Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	// Find admin
	admin, err := s.adminRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find admin", 500)
	}

	if admin == nil {
		return nil, apperrors.ErrInvalidCredentials
	}

	// Check if account is active
	if !admin.IsActive {
		return nil, apperrors.New("ACCOUNT_INACTIVE", "Account is inactive", 403)
	}

	// Check if account is locked
	if admin.LockedUntil != nil && admin.LockedUntil.After(time.Now()) {
		return nil, apperrors.New("ACCOUNT_LOCKED", "Account is temporarily locked", 403)
	}

	// Verify password
	valid, err := crypto.VerifyPassword(req.Password, admin.PasswordHash)
	if err != nil {
		return nil, apperrors.Wrap(err, "PASSWORD_ERROR", "Failed to verify password", 500)
	}

	if !valid {
		// Increment failed attempts
		_ = s.adminRepo.IncrementFailedAttempts(ctx, admin.ID)
		
		// Lock account after 5 failed attempts
		if admin.FailedAttempts >= 4 {
			lockUntil := time.Now().Add(30 * time.Minute)
			_ = s.adminRepo.LockAccount(ctx, admin.ID, lockUntil)
		}

		return nil, apperrors.ErrInvalidCredentials
	}

	// Generate access token
	accessToken, err := s.pasetoMaker.CreateAccessToken(
		admin.ID,
		admin.Email,
		admin.Role,
		admin.Permissions,
		s.cfg.Auth.AccessTokenExpires,
		"admin_access",
	)
	if err != nil {
		return nil, apperrors.Wrap(err, "TOKEN_ERROR", "Failed to generate access token", 500)
	}

	// Generate refresh token
	refreshToken, err := crypto.GenerateRandomToken(32)
	if err != nil {
		return nil, apperrors.Wrap(err, "TOKEN_ERROR", "Failed to generate refresh token", 500)
	}

	// Store refresh token
	tokenHash := jwt.HashToken(refreshToken)
	session := &domain.AdminSession{
		AdminID:    admin.ID,
		TokenHash:  tokenHash,
		DeviceName: &req.DeviceName,
		IPAddress:  &req.IPAddress,
		UserAgent:  &req.UserAgent,
		IsActive:   true,
		ExpiresAt:  time.Now().Add(s.cfg.Auth.RefreshTokenExpires),
	}

	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, apperrors.Wrap(err, "SESSION_ERROR", "Failed to create session", 500)
	}

	// Update last login
	if err := s.adminRepo.UpdateLastLogin(ctx, admin.ID, req.IPAddress); err != nil {
		// Log but don't fail
	}

	// Log activity
	_ = s.logRepo.Create(ctx, &domain.ActivityLog{
		AdminID:    &admin.ID,
		Action:     "admin_login",
		EntityType: strPtr("admin"),
		EntityID:   &admin.ID,
		Details:    domain.JSONMap{"email": admin.Email},
		IPAddress:  &req.IPAddress,
	})

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.cfg.Auth.AccessTokenExpires.Seconds()),
		Admin:        admin,
	}, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (string, error) {
	// Hash token
	tokenHash := jwt.HashToken(refreshToken)

	// Find session
	session, err := s.sessionRepo.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		return "", apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find session", 500)
	}

	if session == nil {
		return "", apperrors.New("INVALID_REFRESH_TOKEN", "Invalid or expired refresh token", 401)
	}

	// Get admin
	admin, err := s.adminRepo.FindByID(ctx, session.AdminID)
	if err != nil {
		return "", apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find admin", 500)
	}

	if admin == nil || !admin.IsActive {
		return "", apperrors.New("ADMIN_NOT_FOUND", "Admin not found or inactive", 401)
	}

	// Generate new access token
	accessToken, err := s.pasetoMaker.CreateAccessToken(
		admin.ID,
		admin.Email,
		admin.Role,
		admin.Permissions,
		s.cfg.Auth.AccessTokenExpires,
		"admin_access",
	)
	if err != nil {
		return "", apperrors.Wrap(err, "TOKEN_ERROR", "Failed to generate access token", 500)
	}

	// Update last used
	_ = s.sessionRepo.UpdateLastUsed(ctx, session.ID)

	return accessToken, nil
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	tokenHash := jwt.HashToken(refreshToken)
	session, err := s.sessionRepo.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		return err
	}

	if session != nil {
		return s.sessionRepo.Revoke(ctx, session.ID)
	}

	return nil
}

func (s *AuthService) LogoutAll(ctx context.Context, adminID int64) error {
	return s.sessionRepo.RevokeAllForAdmin(ctx, adminID)
}

func (s *AuthService) GetActiveSessions(ctx context.Context, adminID int64) ([]*domain.AdminSession, error) {
	return s.sessionRepo.GetActiveSessions(ctx, adminID)
}

func (s *AuthService) RevokeSession(ctx context.Context, sessionID int64) error {
	return s.sessionRepo.Revoke(ctx, sessionID)
}

func (s *AuthService) ChangePassword(ctx context.Context, adminID int64, currentPassword, newPassword string) error {
	// Get admin
	admin, err := s.adminRepo.FindByID(ctx, adminID)
	if err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find admin", 500)
	}

	if admin == nil {
		return apperrors.ErrNotFound
	}

	// Verify current password
	valid, err := crypto.VerifyPassword(currentPassword, admin.PasswordHash)
	if err != nil {
		return apperrors.Wrap(err, "PASSWORD_ERROR", "Failed to verify password", 500)
	}

	if !valid {
		return apperrors.New("INVALID_PASSWORD", "Current password is incorrect", 401)
	}

	// Hash new password
	newHash, err := crypto.HashPassword(newPassword, nil)
	if err != nil {
		return apperrors.Wrap(err, "PASSWORD_ERROR", "Failed to hash password", 500)
	}

	// Update password
	if err := s.adminRepo.UpdatePassword(ctx, adminID, newHash); err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to update password", 500)
	}

	// Revoke all sessions (force re-login)
	_ = s.sessionRepo.RevokeAllForAdmin(ctx, adminID)

	return nil
}

func strPtr(s string) *string {
	return &s
}