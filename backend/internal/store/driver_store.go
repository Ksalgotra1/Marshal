package store

import (
	"context"
	"fmt"
	"strconv"

	"github.com/Ksalgotra1/Marshal/internal/models"
)

type DriverFilter struct {
	Limit  int
	Offset int
}

// DriverStore handles driver CRUD operations.
type DriverStore struct{ DB DBTX }

// Register creates a new driver. Returns the driver's UUID.
func (s *DriverStore) Register(ctx context.Context, name string, telegramID int64) (string, error) {
	var id string
	err := s.DB.QueryRow(ctx, `
		INSERT INTO drivers (name, telegram_id)
		VALUES ($1, $2)
		ON CONFLICT (telegram_id) DO UPDATE SET name = EXCLUDED.name
		RETURNING id
	`, name, telegramID).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("DriverStore.Register: %w", err)
	}
	return id, nil
}

// List returns drivers ordered by newest first.
func (s *DriverStore) List(ctx context.Context, f DriverFilter) ([]models.Driver, error) {
	if f.Limit <= 0 || f.Limit > 100 {
		f.Limit = 20
	}
	if f.Offset < 0 {
		f.Offset = 0
	}

	query := `
			SELECT id, name, telegram_id, telegram_chat, status, created_at
			FROM drivers ORDER BY created_at DESC
			LIMIT $1 OFFSET $2`
	rows, err := s.DB.Query(ctx, query, f.Limit, f.Offset)
	if err != nil {
		return nil, fmt.Errorf("DriverStore.List: %w", err)
	}
	defer rows.Close()

	var drivers []models.Driver
	for rows.Next() {
		var d models.Driver
		if err := rows.Scan(&d.ID, &d.Name, &d.TelegramID, &d.TelegramChat, &d.Status, &d.CreatedAt); err != nil {
			return nil, err
		}
		drivers = append(drivers, d)
	}
	return drivers, nil
}

func ParseOffset(raw string) int {
	offset, _ := strconv.Atoi(raw)
	if offset < 0 {
		return 0
	}
	return offset
}

// GetByID fetches a single driver.
func (s *DriverStore) GetByID(ctx context.Context, id string) (*models.Driver, error) {
	var d models.Driver
	err := s.DB.QueryRow(ctx, `
		SELECT id, name, telegram_id, telegram_chat, status, created_at
		FROM drivers WHERE id = $1
	`, id).Scan(&d.ID, &d.Name, &d.TelegramID, &d.TelegramChat, &d.Status, &d.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("DriverStore.GetByID: %w", err)
	}
	return &d, nil
}

// SetStatus updates a driver's availability status (online/offline/busy).
func (s *DriverStore) SetStatus(ctx context.Context, id, status string) error {
	_, err := s.DB.Exec(ctx, `UPDATE drivers SET status = $1 WHERE id = $2`, status, id)
	return err
}

// GetByTelegramID fetches a single driver by Telegram ID.
func (s *DriverStore) GetByTelegramID(ctx context.Context, telegramID int64) (*models.Driver, error) {
	var d models.Driver
	err := s.DB.QueryRow(ctx, `
		SELECT id, name, telegram_id, telegram_chat, status, created_at
		FROM drivers WHERE telegram_id = $1
	`, telegramID).Scan(&d.ID, &d.Name, &d.TelegramID, &d.TelegramChat, &d.Status, &d.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("DriverStore.GetByTelegramID: %w", err)
	}
	return &d, nil
}

// SetTelegramChat updates the driver's private chat ID.
func (s *DriverStore) SetTelegramChat(ctx context.Context, telegramID int64, chatID int64) error {
	_, err := s.DB.Exec(ctx, `UPDATE drivers SET telegram_chat = $1 WHERE telegram_id = $2`, chatID, telegramID)
	return err
}

// CountOnline returns the number of online drivers.
func (s *DriverStore) CountOnline(ctx context.Context) (int, error) {
	var count int
	err := s.DB.QueryRow(ctx, `SELECT COUNT(*) FROM drivers WHERE status = 'online'`).Scan(&count)
	return count, err
}
