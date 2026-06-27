package config

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	Port                string
	DatabaseURL         string
	AllowedOrigin       string
	LogFormat           string
	TelegramBotToken      string
	TelegramDriverGroup   string
	TelegramWebhookURL    string
	TelegramWebhookSecret string
	DriverPresenceTTL     string // minutes, parsed where used
	AdminAPIKey           string
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
		TelegramBotToken:      os.Getenv("TELEGRAM_BOT_TOKEN"),
		TelegramDriverGroup:   os.Getenv("TELEGRAM_DRIVER_GROUP_ID"),
		TelegramWebhookURL:    os.Getenv("TELEGRAM_WEBHOOK_URL"),
		TelegramWebhookSecret: os.Getenv("TELEGRAM_WEBHOOK_SECRET"),
		DriverPresenceTTL:     getEnvOr("DRIVER_PRESENCE_TTL_MINUTES", "15"),
		AdminAPIKey:           os.Getenv("ADMIN_API_KEY"),
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

// RunMigrations runs all .up.sql files in migrationsDir lexicographically.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool, migrationsDir string) error {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	// Check if golang-migrate is managing the schema
	var version int
	if err := pool.QueryRow(ctx, "SELECT version FROM schema_migrations LIMIT 1").Scan(&version); err == nil {
		slog.Info("golang-migrate is managing schema, skipping internal migration runner")
		return nil
	}

	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	appliedCount := 0

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".up.sql") {
			continue
		}

		filename := entry.Name()

		var exists string
		err := pool.QueryRow(ctx, "SELECT filename FROM schema_migrations WHERE filename = $1", filename).Scan(&exists)
		if err == nil {
			// Already applied
			continue
		}

		content, err := os.ReadFile(filepath.Join(migrationsDir, filename))
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("failed to begin transaction for %s: %w", filename, err)
		}

		if _, err := tx.Exec(ctx, string(content)); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("failed to execute migration %s: %w", filename, err)
		}

		if _, err := tx.Exec(ctx, "INSERT INTO schema_migrations (filename) VALUES ($1)", filename); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("failed to record migration %s: %w", filename, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", filename, err)
		}

		slog.Info("migration applied", "file", filename)
		appliedCount++
	}

	if appliedCount == 0 {
		slog.Info("all migrations up to date")
	}

	return nil
}
