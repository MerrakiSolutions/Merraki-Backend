package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/merraki/merraki-backend/internal/config"
	apperrors "github.com/merraki/merraki-backend/internal/pkg/errors"
	"github.com/merraki/merraki-backend/internal/pkg/jwt"
	"github.com/merraki/merraki-backend/internal/pkg/response"
)

func AdminAuth(cfg *config.Config) fiber.Handler {
	pasetoMaker, err := jwt.NewPasetoMaker(cfg.Auth.PasetoKey)
	if err != nil {
		panic("Failed to initialize PASETO maker: " + err.Error())
	}

	return func(c *fiber.Ctx) error {
		// 🔐 Get access token from cookie
		token := c.Cookies("admin_access_token")
		if token == "" {
			return response.Error(c, apperrors.ErrUnauthorized)
		}

		// 🔍 Verify token
		claims, err := pasetoMaker.VerifyToken(token)
		if err != nil {
			return response.Error(c, apperrors.New("INVALID_TOKEN", err.Error(), 401))
		}

		// 🧠 Ensure correct token type
		if claims.Type != "admin_access" {
			return response.Error(c, apperrors.New("INVALID_TOKEN_TYPE", "Invalid token type", 401))
		}

		// 📌 Store claims in context
		c.Locals("admin_id", claims.AdminID)
		c.Locals("admin_email", claims.Email)
		c.Locals("admin_role", claims.Role)
		c.Locals("admin_permissions", claims.Permissions)

		return c.Next()
	}
}

func RequirePermission(permission string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		permissions, ok := c.Locals("admin_permissions").(map[string]interface{})
		if !ok {
			return response.Error(c, apperrors.ErrForbidden)
		}

		// Super admin has all permissions
		if all, ok := permissions["all"].(bool); ok && all {
			return c.Next()
		}

		// Check specific permission
		if perm, ok := permissions[permission].(bool); ok && perm {
			return c.Next()
		}

		return response.Error(c, apperrors.New("INSUFFICIENT_PERMISSIONS", "Insufficient permissions", 403))
	}
}

func GetAdminID(c *fiber.Ctx) int64 {
	adminID, _ := c.Locals("admin_id").(int64)
	return adminID
}

func GetAdminEmail(c *fiber.Ctx) string {
	email, _ := c.Locals("admin_email").(string)
	return email
}

func GetAdminRole(c *fiber.Ctx) string {
	role, _ := c.Locals("admin_role").(string)
	return role
}