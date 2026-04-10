package routes

import (
	"github.com/gofiber/fiber/v2"
	publicHandlers "github.com/merraki/merraki-backend/internal/handler/public"
)

// ============================================================================
// PUBLIC HANDLERS STRUCT
// ============================================================================

type PublicHandlers struct {
	Template   *publicHandlers.TemplateHandler
	Order      *publicHandlers.OrderHandler
	Checkout   *publicHandlers.CheckoutHandler
	Download   *publicHandlers.DownloadHandler
	Blog       *publicHandlers.BlogHandler
	Newsletter *publicHandlers.NewsletterHandler
	Contact    *publicHandlers.ContactHandler
	Utility    *publicHandlers.UtilityHandler
}

// ============================================================================
// SETUP PUBLIC ROUTES
// ============================================================================

func SetupPublicRoutes(api fiber.Router, handlers *PublicHandlers) {
	public := api.Group("/public")

	// ========================================================================
	// TEMPLATES - Product catalog
	// ========================================================================
	templates := public.Group("/templates")
	{
		templates.Get("/", handlers.Template.GetAllTemplates)
		// FIX: static paths before /:slug
		templates.Get("/search", handlers.Template.SearchTemplates)
		templates.Get("/featured", handlers.Template.GetFeaturedTemplates)
		templates.Get("/bestsellers", handlers.Template.GetBestsellers)
		templates.Get("/new", handlers.Template.GetNewTemplates)
		templates.Get("/by-id/:id", handlers.Template.GetTemplateByID)
		// parameterized last
		templates.Get("/:slug", handlers.Template.GetTemplateBySlug)
	}

	// ========================================================================
	// CATEGORIES
	// ========================================================================
	categories := public.Group("/categories")
	{
		categories.Get("/", handlers.Template.GetCategories)
		categories.Get("/:slug", handlers.Template.GetCategoryBySlug)
		categories.Get("/:slug/templates", handlers.Template.GetTemplatesByCategory)
	}

	// ========================================================================
	// TAGS
	// ========================================================================
	tags := public.Group("/tags")
	{
		tags.Get("/:tag/templates", handlers.Template.GetTemplatesByTag)
	}

	// ========================================================================
	// CHECKOUT
	// ========================================================================
	checkout := public.Group("/checkout")
	{
		checkout.Post("/create-order", handlers.Checkout.CreateOrder)
		checkout.Post("/initiate-payment", handlers.Checkout.InitiatePayment)
		checkout.Post("/verify-payment", handlers.Checkout.VerifyPayment)
	}

	// ========================================================================
	// WEBHOOKS
	// ========================================================================
	webhooks := public.Group("/webhooks")
	{
		webhooks.Post("/razorpay", handlers.Checkout.HandleWebhook)
	}

	// ========================================================================
	// ORDERS
	// ========================================================================
	orders := public.Group("/orders")
	{
		// FIX: static paths before /:id
		orders.Get("/lookup", handlers.Order.LookupOrder)
		orders.Get("/by-email", handlers.Order.GetOrdersByEmail)
		// parameterized last
		orders.Get("/:id", handlers.Order.GetOrderByID)
	}

	// ========================================================================
	// DOWNLOADS
	// ========================================================================
	download := public.Group("/download")
	{
		// FIX: static paths before parameterized (by-email before any future /:id)
		download.Get("/by-email", handlers.Download.GetDownloadsByEmail)
		download.Post("/info", handlers.Download.GetDownloadInfo)
		download.Get("/", handlers.Download.InitiateDownload)
	}

	// ========================================================================
	// BLOG
	// ========================================================================
	blog := public.Group("/blog")
	{
		blog.Get("/posts", handlers.Blog.GetAllPosts)
		blog.Get("/posts/search", handlers.Blog.SearchPosts) // static before /:slug ✅
		blog.Get("/posts/:slug", handlers.Blog.GetPostBySlug)
		blog.Get("/authors", handlers.Blog.GetAllAuthors)
		blog.Get("/authors/:slug", handlers.Blog.GetAuthorBySlug)
		blog.Get("/categories", handlers.Blog.GetAllCategories)
		blog.Get("/categories/:slug", handlers.Blog.GetCategoryBySlug)
	}

	// ========================================================================
	// NEWSLETTER
	// ========================================================================
	newsletter := public.Group("/newsletter")
	{
		newsletter.Post("/subscribe", handlers.Newsletter.Subscribe)
		newsletter.Post("/unsubscribe", handlers.Newsletter.Unsubscribe)
		newsletter.Get("/unsubscribe", handlers.Newsletter.UnsubscribeGET)
	}

	// ========================================================================
	// CONTACT
	// ========================================================================
	public.Post("/contact", handlers.Contact.Create)
}