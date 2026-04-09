package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Auth     AuthConfig
	Storage  StorageConfig
	Payment  PaymentConfig
	Email    EmailConfig
	Frontend FrontendConfig
	CORS     CORSConfig
	Security SecurityConfig
	Logging  LoggingConfig
}

type ServerConfig struct {
	Port        int
	WorkerPort  int
	Environment string
	APIVersion  string
}

type DatabaseConfig struct {
	Host           string
	Port           int
	User           string
	Password       string
	Name           string
	SSLMode        string
	MaxConnections int
	MaxIdle        int
	MaxLifetime    time.Duration
}

type RedisConfig struct {
	Host      string
	Port      int
	Password  string
	DB        int
	MaxIdle   int
	MaxActive int
}

type AuthConfig struct {
	PasetoKey           string
	AccessTokenExpires  time.Duration
	RefreshTokenExpires time.Duration
	CookieSecure        bool
	CookieSameSite      string
}

type StorageConfig struct {
	Provider         string
	CloudinaryName   string
	CloudinaryKey    string
	CloudinarySecret string
	CloudinaryFolder string
}

type PaymentConfig struct {
	RazorpayKeyID         string
	RazorpayKeySecret     string
	RazorpayWebhookSecret string
	Razorpay              RazorpayConfig `mapstructure:"razorpay"`
}

type EmailConfig struct {
	Provider     string
	SendGridKey  string
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	FromEmail    string
	FromName     string
}

type RazorpayConfig struct {
	KeyID         string `mapstructure:"key_id"`
	KeySecret     string `mapstructure:"key_secret"`
	WebhookSecret string `mapstructure:"webhook_secret"`
	BaseURL       string `mapstructure:"base_url"`
}

type FrontendConfig struct {
	URL      string
	AdminURL string
}

type CORSConfig struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
}

type SecurityConfig struct {
	Argon2Time      uint32
	Argon2Memory    uint32
	Argon2Threads   uint8
	Argon2KeyLength uint32
}

type LoggingConfig struct {
	Level  string
	Format string
	Output string
}

func Load() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {

	}
	cfg := &Config{
		Server: ServerConfig{
			Port:        viper.GetInt("PORT"),
			WorkerPort:  viper.GetInt("WORKER_PORT"),
			Environment: viper.GetString("ENV"),
			APIVersion:  viper.GetString("API_VERSION"),
		},
		Database: DatabaseConfig{
			Host:           viper.GetString("DB_HOST"),
			Port:           viper.GetInt("DB_PORT"),
			User:           viper.GetString("DB_USER"),
			Password:       viper.GetString("DB_PASSWORD"),
			Name:           viper.GetString("DB_NAME"),
			SSLMode:        viper.GetString("DB_SSL_MODE"),
			MaxConnections: viper.GetInt("DB_MAX_CONNECTIONS"),
			MaxIdle:        viper.GetInt("DB_MAX_IDLE_CONNECTIONS"),
			MaxLifetime:    viper.GetDuration("DB_MAX_LIFETIME_MINUTES") * time.Minute,
		},
		Redis: RedisConfig{
			Host:      viper.GetString("REDIS_HOST"),
			Port:      viper.GetInt("REDIS_PORT"),
			Password:  viper.GetString("REDIS_PASSWORD"),
			DB:        viper.GetInt("REDIS_DB"),
			MaxIdle:   viper.GetInt("REDIS_MAX_IDLE"),
			MaxActive: viper.GetInt("REDIS_MAX_ACTIVE"),
		},
		Auth: AuthConfig{
			PasetoKey:           viper.GetString("PASETO_SYMMETRIC_KEY"),
			AccessTokenExpires:  viper.GetDuration("ACCESS_TOKEN_EXPIRES"),
			RefreshTokenExpires: viper.GetDuration("REFRESH_TOKEN_EXPIRES"),
			CookieSecure:        viper.GetBool("COOKIE_SECURE"),
			CookieSameSite:      viper.GetString("COOKIE_SAME_SITE"),
		},
		Storage: StorageConfig{
			Provider:         viper.GetString("STORAGE_PROVIDER"),
			CloudinaryName:   viper.GetString("CLOUDINARY_CLOUD_NAME"),
			CloudinaryKey:    viper.GetString("CLOUDINARY_API_KEY"),
			CloudinarySecret: viper.GetString("CLOUDINARY_API_SECRET"),
			CloudinaryFolder: viper.GetString("CLOUDINARY_FOLDER"),
		},
		Payment: PaymentConfig{
			RazorpayKeyID:         viper.GetString("RAZORPAY_KEY_ID"),
			RazorpayKeySecret:     viper.GetString("RAZORPAY_KEY_SECRET"),
			RazorpayWebhookSecret: viper.GetString("RAZORPAY_WEBHOOK_SECRET"),
			Razorpay: RazorpayConfig{
				KeyID:         viper.GetString("RAZORPAY_KEY_ID"),
				KeySecret:     viper.GetString("RAZORPAY_KEY_SECRET"),
				WebhookSecret: viper.GetString("RAZORPAY_WEBHOOK_SECRET"),
				BaseURL:       "https://api.razorpay.com/v1",
			},
		},
		Email: EmailConfig{
			Provider:     viper.GetString("EMAIL_PROVIDER"),
			SendGridKey:  viper.GetString("SENDGRID_API_KEY"),
			SMTPHost:     viper.GetString("SMTP_HOST"),
			SMTPPort:     viper.GetInt("SMTP_PORT"),
			SMTPUsername: viper.GetString("SMTP_USERNAME"),
			SMTPPassword: viper.GetString("SMTP_PASSWORD"),
			FromEmail:    viper.GetString("EMAIL_FROM"),
			FromName:     viper.GetString("EMAIL_FROM_NAME"),
		},
		Frontend: FrontendConfig{
			URL:      viper.GetString("FRONTEND_URL"),
			AdminURL: viper.GetString("ADMIN_URL"),
		},
		CORS: CORSConfig{
			AllowedOrigins: viper.GetStringSlice("CORS_ALLOWED_ORIGINS"),
			AllowedMethods: viper.GetStringSlice("CORS_ALLOWED_METHODS"),
			AllowedHeaders: viper.GetStringSlice("CORS_ALLOWED_HEADERS"),
		},
		Security: SecurityConfig{
			Argon2Time:      uint32(viper.GetInt("ARGON2_TIME")),
			Argon2Memory:    uint32(viper.GetInt("ARGON2_MEMORY")),
			Argon2Threads:   uint8(viper.GetInt("ARGON2_THREADS")),
			Argon2KeyLength: uint32(viper.GetInt("ARGON2_KEY_LENGTH")),
		},
		Logging: LoggingConfig{
			Level:  viper.GetString("LOG_LEVEL"),
			Format: viper.GetString("LOG_FORMAT"),
			Output: viper.GetString("LOG_OUTPUT"),
		},
	}

	return cfg, nil
}

func (c *Config) GetDSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.Name,
		c.Database.SSLMode,
	)
}

func (c *Config) GetRedisAddr() string {
	return fmt.Sprintf("%s:%d", c.Redis.Host, c.Redis.Port)
}

func (c *Config) IsDevelopment() bool {
	return c.Server.Environment == "development"
}

func (c *Config) IsProduction() bool {
	return c.Server.Environment == "production"
}
