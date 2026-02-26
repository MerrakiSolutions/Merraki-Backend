package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jmoiron/sqlx"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/merraki/merraki-backend/internal/config"
	"github.com/merraki/merraki-backend/internal/pkg/logger"
	"go.uber.org/zap"
)

type Database struct {
	Pool *pgxpool.Pool
	DB   *sqlx.DB
}

func NewDatabase(cfg *config.Config) (*Database, error) {
	// Create pgxpool connection
	poolConfig, err := pgxpool.ParseConfig(cfg.GetDSN())
	if err != nil {
		return nil, fmt.Errorf("unable to parse database config: %w", err)
	}

	poolConfig.MaxConns = int32(cfg.Database.MaxConnections)
	poolConfig.MinConns = int32(cfg.Database.MaxIdle)
	poolConfig.MaxConnLifetime = cfg.Database.MaxLifetime
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	// Also create sqlx connection for complex queries
	db, err := sqlx.Connect("pgx", cfg.GetDSN())
	if err != nil {
		return nil, fmt.Errorf("unable to connect with sqlx: %w", err)
	}

	db.SetMaxOpenConns(cfg.Database.MaxConnections)
	db.SetMaxIdleConns(cfg.Database.MaxIdle)
	db.SetConnMaxLifetime(cfg.Database.MaxLifetime)

	logger.Info("Database connected successfully",
		zap.String("host", cfg.Database.Host),
		zap.Int("port", cfg.Database.Port),
		zap.String("database", cfg.Database.Name),
	)

	return &Database{
		Pool: pool,
		DB:   db,
	}, nil
}

func (d *Database) Close() {
	if d.Pool != nil {
		d.Pool.Close()
	}
	if d.DB != nil {
		_ = d.DB.Close()
	}
}

func (d *Database) Health() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := d.Pool.Ping(ctx); err != nil {
		return err
	}

	return nil
}