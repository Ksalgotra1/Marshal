package config

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	Port                string
	DatabaseURL         string
	AllowedOrigin       string
	LogFormat           string
	TelegramBotToken    string
	TelegramDriverGroup string
	TelegramWebhookURL  string
}

// Load reads environment variables into a Config struct.
// Falls back to .env file in development, never panics on missing .env.
func Load() *Config {
	_ = godotenv.Load() // ignore error — .env is optional in production

	return &Config{
		Port:                getEnvOr("PORT", "8080"),
		DatabaseURL:         getEnvOr("DATABASE_URL", "postgres://marshal:marshal_secret@localhost:5433/marshal_db?sslmode=disable"),
		AllowedOrigin:       getEnvOr("ALLOWED_ORIGIN", "http://localhost:5173"),
		LogFormat:           getEnvOr("LOG_FORMAT", "text"),
		TelegramBotToken:    os.Getenv("TELEGRAM_BOT_TOKEN"),
		TelegramDriverGroup: os.Getenv("TELEGRAM_DRIVER_GROUP_ID"),
		TelegramWebhookURL:  os.Getenv("TELEGRAM_WEBHOOK_URL"),
	}
}

// ConnectDB creates a connection pool to Postgres.
func ConnectDB(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	slog.Info("database connection established")
	return pool, nil
}

// InitLogger sets up the global structured logger.
func InitLogger(format string) {
	var handler slog.Handler
	if format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	}
	slog.SetDefault(slog.New(handler))
}

func getEnvOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
