package admin

import (
	"github.com/gofiber/fiber/v2"
	"github.com/merraki/merraki-backend/internal/middleware"
	"github.com/merraki/merraki-backend/internal/pkg/response"
	"github.com/merraki/merraki-backend/internal/pkg/validator"
	"github.com/merraki/merraki-backend/internal/service"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, err)
	}

	if err := validator.Validate(req); err != nil {
		return response.ValidationError(c, validator.FormatValidationErrors(err))
	}

	loginReq := &service.LoginRequest{
		Email:      req.Email,
		Password:   req.Password,
		IPAddress:  c.IP(),
		UserAgent:  c.Get("User-Agent"),
		DeviceName: c.Get("User-Agent"), // Can be enhanced
	}

	loginResp, err := h.authService.Login(c.Context(), loginReq)
	if err != nil {
		return response.Error(c, err)
	}

	// Access token cookie (short-lived)
	c.Cookie(&fiber.Cookie{
		Name:     "admin_access_token",
		Value:    loginResp.AccessToken,
		HTTPOnly: true,     // prevents JS access
		Secure:   true,     // requires HTTPS
		SameSite: "Strict", // tighter CSRF protection
		Path:     "/",
		MaxAge:   loginResp.ExpiresIn, // e.g., 300 seconds
	})

	// Refresh token cookie (long-lived)
	c.Cookie(&fiber.Cookie{
		Name:     "admin_refresh_token",
		Value:    loginResp.RefreshToken,
		HTTPOnly: true,
		Secure:   true,     // requires HTTPS
		SameSite: "Strict", // tighter CSRF protection
		Path:     "/",
		MaxAge:   7 * 24 * 60 * 60, // 7 days
	})

	return response.Success(c, "Login successful", fiber.Map{
		"access_token": loginResp.AccessToken,
		"token_type":   loginResp.TokenType,
		"expires_in":   loginResp.ExpiresIn,
		"admin":        loginResp.Admin,
	})
}

func (h *AuthHandler) RefreshToken(c *fiber.Ctx) error {
	refreshToken := c.Cookies("admin_refresh_token")
	if refreshToken == "" {
		return response.Error(c, fiber.NewError(401, "Refresh token not found"))
	}

	accessToken, err := h.authService.RefreshToken(c.Context(), refreshToken)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, "Token refreshed successfully", fiber.Map{
		"access_token": accessToken,
		"token_type":   "Bearer",
		"expires_in":   300,
	})
}

func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	refreshToken := c.Cookies("admin_refresh_token")
	if refreshToken != "" {
		_ = h.authService.Logout(c.Context(), refreshToken)
	}

	c.ClearCookie("admin_access_token")
	c.ClearCookie("admin_refresh_token")

	return response.Success(c, "Logged out successfully", nil)
}

func (h *AuthHandler) LogoutAll(c *fiber.Ctx) error {
	adminID := middleware.GetAdminID(c)

	if err := h.authService.LogoutAll(c.Context(), adminID); err != nil {
		return response.Error(c, err)
	}

	c.ClearCookie("admin_refresh_token")
	c.ClearCookie("admin_access_token")

	return response.Success(c, "Logged out from all devices successfully", nil)
}

func (h *AuthHandler) GetMe(c *fiber.Ctx) error {
	adminID := middleware.GetAdminID(c)
	email := middleware.GetAdminEmail(c)
	role := middleware.GetAdminRole(c)

	return response.SuccessData(c, fiber.Map{
		"admin": fiber.Map{
			"id":    adminID,
			"email": email,
			"role":  role,
		},
	})
}

func (h *AuthHandler) GetSessions(c *fiber.Ctx) error {
	adminID := middleware.GetAdminID(c)

	sessions, err := h.authService.GetActiveSessions(c.Context(), adminID)
	if err != nil {
		return response.Error(c, err)
	}

	return response.SuccessData(c, fiber.Map{
		"sessions": sessions,
	})
}

func (h *AuthHandler) RevokeSession(c *fiber.Ctx) error {
	sessionID, err := c.ParamsInt("sessionId")
	if err != nil {
		return response.Error(c, fiber.NewError(400, "Invalid session ID"))
	}

	if err := h.authService.RevokeSession(c.Context(), int64(sessionID)); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, "Session revoked successfully", nil)
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,password"`
}

func (h *AuthHandler) ChangePassword(c *fiber.Ctx) error {
	var req ChangePasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, err)
	}

	if err := validator.Validate(req); err != nil {
		return response.ValidationError(c, validator.FormatValidationErrors(err))
	}

	adminID := middleware.GetAdminID(c)

	if err := h.authService.ChangePassword(c.Context(), adminID, req.CurrentPassword, req.NewPassword); err != nil {
		return response.Error(c, err)
	}

	c.ClearCookie("admin_refresh_token")

	return response.Success(c, "Password changed successfully. Please login again.", nil)
}

// Add this method to existing auth_handler.go

func (h *AuthHandler) GetLoginHistory(c *fiber.Ctx) error {
	// TODO: Get login history
	return response.SuccessData(c, fiber.Map{
		"history": []interface{}{},
	})
}
