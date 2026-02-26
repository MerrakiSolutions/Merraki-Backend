package main

import (
	"fmt"
	"log"
	"os"

	"github.com/merraki/merraki-backend/internal/config"
	"github.com/merraki/merraki-backend/internal/pkg/logger"
	"github.com/merraki/merraki-backend/internal/repository/postgres"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: migrate [up|down|reset]")
		os.Exit(1)
	}

	command := os.Args[1]

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

	switch command {
	case "up":
		fmt.Println("Running migrations...")
		if err := runMigrationsUp(db); err != nil {
			log.Fatal("Migration failed:", err)
		}
		fmt.Println("✅ Migrations completed successfully!")

	case "down":
		fmt.Println("Rolling back migrations...")
		if err := runMigrationsDown(db); err != nil {
			log.Fatal("Rollback failed:", err)
		}
		fmt.Println("✅ Rollback completed successfully!")

	case "reset":
		fmt.Println("Resetting database...")
		if err := runMigrationsDown(db); err != nil {
			log.Fatal("Rollback failed:", err)
		}
		if err := runMigrationsUp(db); err != nil {
			log.Fatal("Migration failed:", err)
		}
		fmt.Println("✅ Database reset successfully!")

	default:
		fmt.Printf("Unknown command: %s\n", command)
		fmt.Println("Available commands: up, down, reset")
		os.Exit(1)
	}
}

func runMigrationsUp(db *postgres.Database) error {
	// Read and execute migration file
	migrationSQL, err := os.ReadFile("migrations/000001_init_schema.up.sql")
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	if _, err := db.DB.Exec(string(migrationSQL)); err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	return nil
}

func runMigrationsDown(db *postgres.Database) error {
	// Read and execute rollback file
	migrationSQL, err := os.ReadFile("migrations/000001_init_schema.down.sql")
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	if _, err := db.DB.Exec(string(migrationSQL)); err != nil {
		return fmt.Errorf("failed to execute rollback: %w", err)
	}

	return nil
}
