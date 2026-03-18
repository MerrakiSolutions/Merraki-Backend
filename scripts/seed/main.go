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

func strPtr(s string) *string {
	return &s
}


