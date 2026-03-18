package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/merraki/merraki-backend/internal/config"
	adminHandlers "github.com/merraki/merraki-backend/internal/handler/admin"
	"github.com/merraki/merraki-backend/internal/middleware"
)

type AdminHandlers struct {
	Auth         *adminHandlers.AuthHandler
	Dashboard    *adminHandlers.DashboardHandler
	Order        *adminHandlers.OrderHandler
	Template     *adminHandlers.TemplateHandler
	Category     *adminHandlers.CategoryHandler
	BlogPost     *adminHandlers.BlogPostHandler
	BlogAuthor   *adminHandlers.BlogAuthorHandler
	BlogCategory *adminHandlers.BlogCategoryHandler
	Newsletter   *adminHandlers.NewsletterHandler
	Contact      *adminHandlers.ContactHandler
	AdminUser    *adminHandlers.AdminUserHandler
	Currency     *adminHandlers.CurrencyHandler
}

func SetupAdminRoutes(api fiber.Router, h *AdminHandlers, cfg *config.Config) {
	admin := api.Group("/admin")

	setupAuthRoutes(admin, h)
	protected := admin.Use(middleware.AdminAuth(cfg))

	setupProtectedAuthRoutes(protected, h)
	setupDashboardRoutes(protected, h)
	setupCurrencyRoutes(api, h)
	setupBlogRoutes(protected, h)
	setupOrderRoutes(protected, h)
	setupTemplateRoutes(protected, h)
	setupCategoryRoutes(protected, h)
	setupContactRoutes(protected, h)
	setupAdminUserRoutes(protected, h)
	setupGlobalRoutes(protected, h)
}

/* ================= AUTH (PUBLIC) ================= */

func setupAuthRoutes(admin fiber.Router, h *AdminHandlers) {
	auth := admin.Group("/auth")

	auth.Post("/login", h.Auth.Login)
	auth.Post("/refresh", h.Auth.RefreshToken)
}

/* ================= AUTH (PROTECTED) ================= */

func setupProtectedAuthRoutes(protected fiber.Router, h *AdminHandlers) {
	auth := protected.Group("/auth")

	auth.Post("/logout", h.Auth.Logout)
	auth.Post("/logout-all", h.Auth.LogoutAll)
	auth.Get("/me", h.Auth.GetMe)
	auth.Get("/sessions", h.Auth.GetSessions)
	auth.Delete("/sessions/:sessionId", h.Auth.RevokeSession)
	auth.Post("/change-password", h.Auth.ChangePassword)
	auth.Get("/login-history", h.Auth.GetLoginHistory)
}

/* ================= DASHBOARD ================= */

func setupDashboardRoutes(protected fiber.Router, h *AdminHandlers) {
	d := protected.Group("/dashboard")

	d.Get("/summary", h.Dashboard.GetSummary)
	d.Get("/stats", h.Dashboard.GetStats)
	d.Get("/activity", h.Dashboard.GetActivity)
	d.Get("/charts", h.Dashboard.GetCharts)
	d.Get("/notifications", h.Dashboard.GetNotifications)
	d.Put("/notifications/:id/read", h.Dashboard.MarkNotificationRead)
}

/* ================= CURRENCY ================= */

func setupCurrencyRoutes(api fiber.Router, h *AdminHandlers) {
	api.Post("/currency/refresh", h.Currency.RefreshRates)
}

/* ================= BLOG ================= */

func setupBlogRoutes(protected fiber.Router, h *AdminHandlers) {
	blog := protected.Group("/blog")

	// Posts
	posts := blog.Group("/posts")
	posts.Get("/search", h.BlogPost.Search)
	posts.Get("/", h.BlogPost.GetAll)
	posts.Get("/:id", h.BlogPost.GetByID)
	posts.Get("/slug/:slug", h.BlogPost.GetBySlug)
	posts.Post("/", h.BlogPost.Create)
	posts.Put("/:id", h.BlogPost.Update)
	posts.Patch("/:id", h.BlogPost.Patch)
	posts.Delete("/:id", h.BlogPost.Delete)

	// Authors
	authors := blog.Group("/authors")
	authors.Get("/", h.BlogAuthor.GetAll)
	authors.Get("/:id", h.BlogAuthor.GetByID)
	authors.Get("/slug/:slug", h.BlogAuthor.GetBySlug)
	authors.Post("/", h.BlogAuthor.Create)
	authors.Put("/:id", h.BlogAuthor.Update)
	authors.Delete("/:id", h.BlogAuthor.Delete)

	// Categories
	categories := blog.Group("/categories")
	categories.Get("/", h.BlogCategory.GetAll)
	categories.Get("/:id", h.BlogCategory.GetByID)
	categories.Get("/slug/:slug", h.BlogCategory.GetBySlug)
	categories.Post("/", h.BlogCategory.Create)
	categories.Put("/:id", h.BlogCategory.Update)
	categories.Delete("/:id", h.BlogCategory.Delete)
}

/* ================= ORDERS ================= */

func setupOrderRoutes(protected fiber.Router, h *AdminHandlers) {
	o := protected.Group("/orders")

	o.Get("/", h.Order.GetAllOrders)
	o.Get("/pending-review", h.Order.GetPendingReviewOrders)
	o.Get("/:id", h.Order.GetOrderByID)

	o.Post("/:id/approve", h.Order.ApproveOrder)
	o.Post("/:id/reject", h.Order.RejectOrder)
}

/* ================= TEMPLATES ================= */

func setupTemplateRoutes(protected fiber.Router, h *AdminHandlers) {
	t := protected.Group("/templates")

	t.Get("/", h.Template.GetAllTemplates)
	t.Get("/:id", h.Template.GetTemplateByID)

	t.Post("/", h.Template.CreateTemplate)
	t.Put("/:id", h.Template.UpdateTemplate)
	t.Patch("/:id", h.Template.PatchTemplate)
	t.Delete("/:id", h.Template.DeleteTemplate)

	t.Post("/:id/upload-file", h.Template.UploadTemplateFile)

	t.Post("/:id/images", h.Template.AddImage)
	t.Delete("/images/:id", h.Template.DeleteImage)

	t.Post("/:id/features", h.Template.AddFeature)
	t.Delete("/features/:id", h.Template.DeleteFeature)

	t.Put("/:id/tags", h.Template.UpdateTags)
}

/* ================= CATEGORIES ================= */

func setupCategoryRoutes(protected fiber.Router, h *AdminHandlers) {
	c := protected.Group("/categories")

	c.Get("/", h.Category.GetAllCategories)
	c.Get("/:id", h.Category.GetCategoryByID)
	c.Post("/", h.Category.CreateCategory)
	c.Put("/:id", h.Category.UpdateCategory)
	c.Delete("/:id", h.Category.DeleteCategory)
}

/* ================= CONTACTS ================= */

func setupContactRoutes(protected fiber.Router, h *AdminHandlers) {
	c := protected.Group("/contacts")

	c.Get("/analytics", h.Contact.GetAnalytics)
	c.Get("/", h.Contact.GetAll)
	c.Get("/:id", h.Contact.GetByID)
	c.Put("/:id", h.Contact.Update)
	c.Post("/:id/reply", h.Contact.Reply)
	c.Delete("/:id", h.Contact.Delete)
}

/* ================= ADMIN USERS ================= */

func setupAdminUserRoutes(protected fiber.Router, h *AdminHandlers) {
	u := protected.Group("/users")

	u.Get("/", h.AdminUser.GetAll)
	u.Get("/:id", h.AdminUser.GetByID)
	u.Post("/", h.AdminUser.Create)
	u.Put("/:id", h.AdminUser.Update)
	u.Delete("/:id", h.AdminUser.Delete)
}

/* ================= GLOBAL ================= */

func setupGlobalRoutes(protected fiber.Router, h *AdminHandlers) {
	protected.Get("/search", h.Dashboard.GlobalSearch)

	settings := protected.Group("/settings")
	settings.Get("/", h.Dashboard.GetSettings)
	settings.Put("/", h.Dashboard.UpdateSettings)
}