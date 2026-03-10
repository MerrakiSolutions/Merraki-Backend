package routes

import (
	"github.com/gofiber/fiber/v2"
	publicHandlers "github.com/merraki/merraki-backend/internal/handler/public"
)

type PublicHandlers struct {
	Template   *publicHandlers.TemplateHandler
	Order      *publicHandlers.OrderHandler
	Calculator *publicHandlers.CalculatorHandler
	Blog       *publicHandlers.BlogHandler
	Newsletter *publicHandlers.NewsletterHandler
	Contact    *publicHandlers.ContactHandler
	Test       *publicHandlers.TestHandler
	Utility    *publicHandlers.UtilityHandler
}

func SetupPublicRoutes(api fiber.Router, handlers *PublicHandlers) {
	public := api.Group("/public")

	// ========== TEMPLATES ==========
	templates := public.Group("/templates")
	templates.Get("/", handlers.Template.GetAll)
	templates.Get("/featured", handlers.Template.GetFeatured)
	templates.Get("/popular", handlers.Template.GetPopular)
	templates.Get("/search", handlers.Template.Search)
	templates.Get("/:slug", handlers.Template.GetBySlug)

	// Categories
	public.Get("/categories", handlers.Template.GetCategories)

	// ========== ORDERS ==========
	orders := public.Group("/orders")
	orders.Post("/", handlers.Order.Create)
	orders.Post("/verify", handlers.Order.VerifyPayment)
	orders.Get("/lookup", handlers.Order.Lookup)
	orders.Get("/download/:orderNumber", handlers.Order.Download)
	orders.Post("/webhook", handlers.Order.Webhook)

	// ========== CALCULATORS ==========
	calculators := public.Group("/calculators")
	calculators.Post("/valuation", handlers.Calculator.CalculateValuation)
	calculators.Post("/breakeven", handlers.Calculator.CalculateBreakeven)
	calculators.Post("/save", handlers.Calculator.SaveResult)
	calculators.Get("/results", handlers.Calculator.GetResults)
	calculators.Post("/export-pdf", handlers.Calculator.ExportPDF)

	// ========== BLOG ==========
	blog := api.Group("/blog")
	{
		// Posts
		blog.Get("/posts", handlers.Blog.GetAllPosts)
		blog.Get("/posts/search", handlers.Blog.SearchPosts)
		blog.Get("/posts/:slug", handlers.Blog.GetPostBySlug)

		// Categories
		blog.Get("/categories", handlers.Blog.GetAllCategories)
		blog.Get("/categories/:slug", handlers.Blog.GetCategoryBySlug)
		blog.Get("/categories/:slug/posts", handlers.Blog.GetPostsByCategory)

		// Authors
		blog.Get("/authors", handlers.Blog.GetAllAuthors)
		blog.Get("/authors/:slug", handlers.Blog.GetAuthorBySlug)
		blog.Get("/authors/:slug/posts", handlers.Blog.GetPostsByAuthor)

		// Tags
		blog.Get("/tags/:tag/posts", handlers.Blog.GetPostsByTag)
	}

	// ========== NEWSLETTER ==========
	newsletter := public.Group("/newsletter")
	newsletter.Post("/subscribe", handlers.Newsletter.Subscribe)
	newsletter.Post("/unsubscribe", handlers.Newsletter.Unsubscribe)
	newsletter.Get("/unsubscribe", handlers.Newsletter.UnsubscribeGET)

	// ========== CONTACT ==========
	public.Post("/contact", handlers.Contact.Create)

	// ========== TEST ==========
	test := public.Group("/test")
	test.Get("/questions", handlers.Test.GetQuestions)
	test.Post("/submit", handlers.Test.Submit)
	test.Get("/results/:testNumber", handlers.Test.GetResults)

	// ========== UTILITY ==========
	public.Get("/currency/convert", handlers.Utility.CurrencyConvert)
}
