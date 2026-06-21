package store

import (
	"context"
	"fmt"

	"github.com/Ksalgotra1/Marshal/internal/models"
)

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

// List returns all drivers.
func (s *DriverStore) List(ctx context.Context) ([]models.Driver, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT id, name, telegram_id, telegram_chat, status, created_at
		FROM drivers ORDER BY created_at DESC
	`)
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
