package main

import (
	"context"
	"fmt"
	"log"

	"github.com/merraki/merraki-backend/internal/config"
	"github.com/merraki/merraki-backend/internal/domain"
	"github.com/merraki/merraki-backend/internal/pkg/crypto"
	"github.com/merraki/merraki-backend/internal/pkg/logger"
	"github.com/merraki/merraki-backend/internal/repository/postgres"
)

func main() {
	fmt.Println("🌱 Seeding database...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// ✅ Initialize logger BEFORE database
	if err := logger.InitLogger(cfg.Logging.Level, cfg.Logging.Format, cfg.Logging.Output); err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}
	defer logger.Sync()


	// Connect to database
	db, err := postgres.NewDatabase(cfg)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Seed admin
	adminRepo := postgres.NewAdminRepository(db)
	if err := seedAdmin(ctx, adminRepo); err != nil {
		log.Fatal("Failed to seed admin:", err)
	}

	// Seed categories
	categoryRepo := postgres.NewCategoryRepository(db)
	if err := seedTemplateCategories(ctx, categoryRepo); err != nil {
		log.Fatal("Failed to seed template categories:", err)
	}
	if err := seedBlogCategories(ctx, categoryRepo); err != nil {
		log.Fatal("Failed to seed blog categories:", err)
	}

	// Seed templates
	templateRepo := postgres.NewTemplateRepository(db)
	if err := seedTemplates(ctx, templateRepo); err != nil {
		log.Fatal("Failed to seed templates:", err)
	}

	fmt.Println("✅ Database seeded successfully!")
}

func seedAdmin(ctx context.Context, repo *postgres.AdminRepository) error {
	fmt.Println("Seeding admin user...")

	// Check if admin exists
	existing, _ := repo.FindByEmail(ctx, "admin@merraki.com")
	if existing != nil {
		fmt.Println("  ⏭️  Admin already exists, skipping")
		return nil
	}

	// Hash password
	passwordHash, err := crypto.HashPassword("Admin@2025", nil)
	if err != nil {
		return err
	}

	admin := &domain.Admin{
		Email:        "admin@merraki.com",
		Name:         "Super Admin",
		PasswordHash: passwordHash,
		Role:         "super_admin",
		Permissions:  domain.JSONMap{"all": true},
		IsActive:     true,
	}

	if err := repo.Create(ctx, admin); err != nil {
		return err
	}

	fmt.Println("  ✅ Admin user created (email: admin@merraki.com, password: Admin@2025)")
	return nil
}

func seedTemplateCategories(ctx context.Context, repo *postgres.CategoryRepository) error {
	fmt.Println("Seeding template categories...")

	categories := []*domain.TemplateCategory{
		{
			Slug:         "business-planning",
			Name:         "Business Planning",
			Description:  strPtr("Strategic business planning templates"),
			IconName:     strPtr("briefcase"),
			ColorHex:     strPtr("#FF5733"),
			DisplayOrder: 1,
			IsActive:     true,
		},
		{
			Slug:         "financial-models",
			Name:         "Financial Models",
			Description:  strPtr("Financial modeling and forecasting templates"),
			IconName:     strPtr("calculator"),
			ColorHex:     strPtr("#28A745"),
			DisplayOrder: 2,
			IsActive:     true,
		},
		{
			Slug:         "pitch-decks",
			Name:         "Pitch Decks",
			Description:  strPtr("Investor pitch deck templates"),
			IconName:     strPtr("presentation"),
			ColorHex:     strPtr("#007BFF"),
			DisplayOrder: 3,
			IsActive:     true,
		},
		{
			Slug:         "legal-docs",
			Name:         "Legal Documents",
			Description:  strPtr("Legal templates and agreements"),
			IconName:     strPtr("file-text"),
			ColorHex:     strPtr("#6C757D"),
			DisplayOrder: 4,
			IsActive:     true,
		},
	}

	for _, cat := range categories {
		existing, _ := repo.GetTemplateCategoryBySlug(ctx, cat.Slug)
		if existing != nil {
			fmt.Printf("  ⏭️  Category '%s' already exists, skipping\n", cat.Name)
			continue
		}

		if err := repo.CreateTemplateCategory(ctx, cat); err != nil {
			return err
		}
		fmt.Printf("  ✅ Created category: %s\n", cat.Name)
	}

	return nil
}

func seedBlogCategories(ctx context.Context, repo *postgres.CategoryRepository) error {
	fmt.Println("Seeding blog categories...")

	categories := []*domain.BlogCategory{
		{
			Slug:         "financial-planning",
			Name:         "Financial Planning",
			Description:  strPtr("Tips and guides on financial planning"),
			DisplayOrder: 1,
			IsActive:     true,
		},
		{
			Slug:         "fundraising",
			Name:         "Fundraising",
			Description:  strPtr("Fundraising strategies and insights"),
			DisplayOrder: 2,
			IsActive:     true,
		},
		{
			Slug:         "operations",
			Name:         "Operations",
			Description:  strPtr("Business operations best practices"),
			DisplayOrder: 3,
			IsActive:     true,
		},
	}

	for _, cat := range categories {
		if err := repo.CreateBlogCategory(ctx, cat); err != nil {
			// Skip if already exists
			continue
		}
		fmt.Printf("  ✅ Created blog category: %s\n", cat.Name)
	}

	return nil
}

func seedTemplates(ctx context.Context, repo *postgres.TemplateRepository) error {
	fmt.Println("Seeding templates...")

	templates := []*domain.Template{
		{
			Slug:        "comprehensive-business-plan",
			Title:       "Comprehensive Business Plan Template",
			Description: strPtr("A complete business plan template with financial projections"),
			PriceINR:    99900, // ₹999
			CategoryID:  1,
			Tags:        []string{"business", "planning", "startup"},
			Status:      "active",
			IsFeatured:  true,
			FileURL:     "https://example.com/templates/business-plan.xlsx",
		},
	}

	for _, tmpl := range templates {
		existing, _ := repo.FindBySlug(ctx, tmpl.Slug)
		if existing != nil {
			fmt.Printf("  ⏭️  Template '%s' already exists, skipping\n", tmpl.Title)
			continue
		}

		if err := repo.Create(ctx, tmpl); err != nil {
			return err
		}
		fmt.Printf("  ✅ Created template: %s\n", tmpl.Title)
	}

	return nil
}

func strPtr(s string) *string {
	return &s
}


