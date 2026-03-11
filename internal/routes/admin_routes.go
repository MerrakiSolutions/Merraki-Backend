package routes

import (
	"github.com/gofiber/fiber/v2"
	adminHandlers "github.com/merraki/merraki-backend/internal/handler/admin"
	"github.com/merraki/merraki-backend/internal/middleware"
	"github.com/merraki/merraki-backend/internal/config"
)

type AdminHandlers struct {
	Auth           *adminHandlers.AuthHandler
	Dashboard      *adminHandlers.DashboardHandler
	Template       *adminHandlers.TemplateHandler
	Order          *adminHandlers.OrderHandler
	BlogPost       *adminHandlers.BlogPostHandler
	BlogAuthor     *adminHandlers.BlogAuthorHandler
	BlogCategory   *adminHandlers.BlogCategoryHandler
	Newsletter     *adminHandlers.NewsletterHandler
	Contact        *adminHandlers.ContactHandler
	Test           *adminHandlers.TestHandler
	Calculator     *adminHandlers.CalculatorHandler
	AdminUser      *adminHandlers.AdminUserHandler
}

func SetupAdminRoutes(api fiber.Router, handlers *AdminHandlers, cfg *config.Config) {
	admin := api.Group("/admin")

	// ========== AUTH (No authentication required) ==========
	auth := admin.Group("/auth")
	auth.Post("/login", handlers.Auth.Login)
	auth.Post("/refresh", handlers.Auth.RefreshToken)

	// ========== PROTECTED ROUTES ==========
	protected := admin.Use(middleware.AdminAuth(cfg))

	// Auth Management
	authRoutes := protected.Group("/auth")
	authRoutes.Post("/logout", handlers.Auth.Logout)
	authRoutes.Post("/logout-all", handlers.Auth.LogoutAll)
	authRoutes.Get("/me", handlers.Auth.GetMe)
	authRoutes.Get("/sessions", handlers.Auth.GetSessions)
	authRoutes.Delete("/sessions/:sessionId", handlers.Auth.RevokeSession)
	authRoutes.Post("/change-password", handlers.Auth.ChangePassword)
	authRoutes.Get("/login-history", handlers.Auth.GetLoginHistory)

	// ========== DASHBOARD ==========
	dashboard := protected.Group("/dashboard")
	dashboard.Get("/summary", handlers.Dashboard.GetSummary)
	dashboard.Get("/stats", handlers.Dashboard.GetStats)
	dashboard.Get("/activity", handlers.Dashboard.GetActivity)
	dashboard.Get("/charts", handlers.Dashboard.GetCharts)
	dashboard.Get("/notifications", handlers.Dashboard.GetNotifications)
	dashboard.Put("/notifications/:id/read", handlers.Dashboard.MarkNotificationRead)

	// ========== TEMPLATES ==========
	templates := protected.Group("/templates")
	templates.Get("/analytics", handlers.Template.GetAnalytics)
	templates.Get("/", handlers.Template.GetAll)
	templates.Get("/:id", handlers.Template.GetByID)
	templates.Post("/", handlers.Template.Create)
	templates.Put("/:id", handlers.Template.Update)
	templates.Delete("/:id", handlers.Template.Delete)

	// ========== CATEGORIES FOR TEMPLATES ========== //

	// ========== ORDERS ==========
	orders := protected.Group("/orders")
	orders.Get("/analytics/revenue", handlers.Order.GetRevenueAnalytics)
	orders.Get("/pending", handlers.Order.GetPending)
	orders.Get("/", handlers.Order.GetAll)
	orders.Get("/:id", handlers.Order.GetByID)
	orders.Post("/:id/approve", handlers.Order.Approve)
	orders.Post("/:id/reject", handlers.Order.Reject)

	// ========== BLOG MANAGEMENT (BLOG,CATEGORIES FOR BLOGS , AUTHORS FOR CATEGORIES)==========
	blogManagement := protected.Group("/blog")
	
	// Posts
	posts := blogManagement.Group("/posts")
	posts.Get("/search", handlers.BlogPost.Search)
	posts.Get("/", handlers.BlogPost.GetAll)
	posts.Get("/:id", handlers.BlogPost.GetByID)
	posts.Get("/slug/:slug", handlers.BlogPost.GetBySlug)
	posts.Post("/", handlers.BlogPost.Create)
	posts.Put("/:id", handlers.BlogPost.Update)
	posts.Patch("/:id", handlers.BlogPost.Patch)
	posts.Delete("/:id", handlers.BlogPost.Delete)
	
	// Authors
	authors := blogManagement.Group("/authors")
	authors.Get("/", handlers.BlogAuthor.GetAll)
	authors.Get("/:id", handlers.BlogAuthor.GetByID)
	authors.Get("/slug/:slug", handlers.BlogAuthor.GetBySlug)
	authors.Post("/", handlers.BlogAuthor.Create)
	authors.Put("/:id", handlers.BlogAuthor.Update)
	authors.Delete("/:id", handlers.BlogAuthor.Delete)
	
	// Categories
	blogCategories := blogManagement.Group("/categories")
	blogCategories.Get("/", handlers.BlogCategory.GetAll)
	blogCategories.Get("/:id", handlers.BlogCategory.GetByID)
	blogCategories.Get("/slug/:slug", handlers.BlogCategory.GetBySlug)
	blogCategories.Post("/", handlers.BlogCategory.Create)
	blogCategories.Put("/:id", handlers.BlogCategory.Update)
	blogCategories.Delete("/:id", handlers.BlogCategory.Delete)

	// ========== NEWSLETTER ==========
	//newsletter := protected.Group("/newsletter")
	
	// Subscribers
	//subscribers := newsletter.Group("/subscribers")
	//subscribers.Get("/analytics", handlers.Newsletter.GetSubscriberAnalytics) // Must be before /
	//subscribers.Get("/", handlers.Newsletter.GetAllSubscribers)
	//subscribers.Get("/:id", handlers.Newsletter.GetSubscriberByID)
	//subscribers.Post("/", handlers.Newsletter.AddSubscriber)
	//subscribers.Post("/export", handlers.Newsletter.ExportSubscribers)
	//subscribers.Delete("/", handlers.Newsletter.DeleteAllSubscribers)
	//subscribers.Delete("/:id", handlers.Newsletter.DeleteSubscriber)
	
	// Campaigns
	//campaigns := newsletter.Group("/campaigns")
	//campaigns.Get("/", handlers.Newsletter.GetAllCampaigns)
	//campaigns.Get("/:id", handlers.Newsletter.GetCampaignByID)
	//campaigns.Get("/slug/:slug", handlers.Newsletter.GetCampaignBySlug)
	//campaigns.Post("/", handlers.Newsletter.CreateCampaign)
	//campaigns.Put("/:id", handlers.Newsletter.UpdateCampaign)
	//campaigns.Delete("/:id", handlers.Newsletter.DeleteCampaign)
	//campaigns.Post("/:id/send", handlers.Newsletter.SendCampaign)
	//campaigns.Get("/:id/recipients", handlers.Newsletter.GetCampaignRecipients)

	// ========== CONTACTS ==========
	contacts := protected.Group("/contacts")
	contacts.Get("/analytics", handlers.Contact.GetAnalytics)
	contacts.Get("/", handlers.Contact.GetAll)
	contacts.Get("/:id", handlers.Contact.GetByID)
	contacts.Put("/:id", handlers.Contact.Update)
	contacts.Post("/:id/reply", handlers.Contact.Reply)
	contacts.Delete("/:id", handlers.Contact.Delete)

	// ========== TESTS ==========
	tests := protected.Group("/tests")
	// Questions
	tests.Get("/questions", handlers.Test.GetAllQuestions)
	tests.Get("/questions/:id", handlers.Test.GetQuestionByID)
	tests.Post("/questions", handlers.Test.CreateQuestion)
	tests.Put("/questions/:id", handlers.Test.UpdateQuestion)
	tests.Delete("/questions/:id", handlers.Test.DeleteQuestion)
	// Submissions
	tests.Get("/submissions", handlers.Test.GetAllSubmissions)
	tests.Get("/analytics", handlers.Test.GetAnalytics)
	tests.Post("/export", handlers.Test.Export)

	// ========== CALCULATORS ==========
	calculators := protected.Group("/calculators")
	calculators.Get("/analytics", handlers.Calculator.GetAnalytics)
	calculators.Get("/results", handlers.Calculator.GetAll)

	// ========== ADMIN USERS ==========
	adminUsers := protected.Group("/users")
	adminUsers.Get("/", handlers.AdminUser.GetAll)
	adminUsers.Get("/:id", handlers.AdminUser.GetByID)
	adminUsers.Post("/", handlers.AdminUser.Create)
	adminUsers.Put("/:id", handlers.AdminUser.Update)
	adminUsers.Delete("/:id", handlers.AdminUser.Delete)

	// ========== GLOBAL SEARCH ==========
	protected.Get("/search", handlers.Dashboard.GlobalSearch)

	// ========== SETTINGS ==========
	settings := protected.Group("/settings")
	settings.Get("/", handlers.Dashboard.GetSettings)
	settings.Put("/", handlers.Dashboard.UpdateSettings)
}