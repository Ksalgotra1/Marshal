package store

import (
	"context"
	"fmt"
	"time"

	"github.com/Ksalgotra1/Marshal/internal/models"
)

// GroupStore handles ride group CRUD operations.
type GroupStore struct{ DB DBTX }

// Create inserts a new ride group and links its member requests.
// All operations run inside the caller's transaction.
func (s *GroupStore) Create(ctx context.Context, members []models.RideRequest, score float64) (string, error) {
	// Earliest arrive_by among members is the group's deadline
	arriveBy := members[0].ArriveBy
	for _, m := range members[1:] {
		if m.ArriveBy.Before(arriveBy) {
			arriveBy = m.ArriveBy
		}
	}

	var groupID string
	err := s.DB.QueryRow(ctx, `
		INSERT INTO ride_groups (confidence_score, arrive_by)
		VALUES ($1, $2)
		RETURNING id
	`, score, arriveBy).Scan(&groupID)
	if err != nil {
		return "", fmt.Errorf("GroupStore.Create: %w", err)
	}

	for _, m := range members {
		if _, err := s.DB.Exec(ctx,
			`INSERT INTO group_members (request_id, group_id) VALUES ($1, $2)`,
			m.ID, groupID,
		); err != nil {
			return "", fmt.Errorf("GroupStore.Create member insert: %w", err)
		}
		if _, err := s.DB.Exec(ctx,
			`UPDATE ride_requests SET status = 'grouped', updated_at = NOW() WHERE id = $1`,
			m.ID,
		); err != nil {
			return "", fmt.Errorf("GroupStore.Create status update: %w", err)
		}
	}

	return groupID, nil
}

// ListOpen returns groups still accepting members (status=grouped, no driver yet).
func (s *GroupStore) ListOpen(ctx context.Context) ([]models.RideGroup, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT id, status, confidence_score, arrive_by, expected_departure, driver_id,
		       dispatch_attempts, telegram_msg_id, created_at, updated_at
		FROM ride_groups
		WHERE status = 'grouped' AND driver_id IS NULL AND arrive_by > NOW()
		ORDER BY confidence_score DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("GroupStore.ListOpen: %w", err)
	}
	defer rows.Close()
	return scanGroups(rows)
}

// ListAll returns every group (for admin view), highest score first.
func (s *GroupStore) ListAll(ctx context.Context) ([]models.RideGroup, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT id, status, confidence_score, arrive_by, expected_departure, driver_id,
		       dispatch_attempts, telegram_msg_id, created_at, updated_at
		FROM ride_groups
		ORDER BY confidence_score DESC, created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("GroupStore.ListAll: %w", err)
	}
	defer rows.Close()
	return scanGroups(rows)
}

// GetByID fetches a single group by ID.
func (s *GroupStore) GetByID(ctx context.Context, id string) (*models.RideGroup, error) {
	var g models.RideGroup
	err := s.DB.QueryRow(ctx, `
		SELECT id, status, confidence_score, arrive_by, expected_departure, driver_id,
		       dispatch_attempts, telegram_msg_id, created_at, updated_at
		FROM ride_groups WHERE id = $1
	`, id).Scan(
		&g.ID, &g.Status, &g.ConfidenceScore, &g.ArriveBy, &g.ExpectedDeparture,
		&g.DriverID, &g.DispatchAttempts, &g.TelegramMsgID, &g.CreatedAt, &g.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("GroupStore.GetByID: %w", err)
	}
	return &g, nil
}

// AddMember joins a ride request to an existing open group.
func (s *GroupStore) AddMember(ctx context.Context, groupID, requestID string) error {
	if _, err := s.DB.Exec(ctx,
		`INSERT INTO group_members (request_id, group_id) VALUES ($1, $2)`,
		requestID, groupID,
	); err != nil {
		return fmt.Errorf("GroupStore.AddMember insert: %w", err)
	}
	if _, err := s.DB.Exec(ctx,
		`UPDATE ride_requests SET status = 'grouped', updated_at = NOW() WHERE id = $1`,
		requestID,
	); err != nil {
		return fmt.Errorf("GroupStore.AddMember update request: %w", err)
	}
	_, err := s.DB.Exec(ctx,
		`UPDATE ride_groups SET updated_at = NOW() WHERE id = $1`,
		groupID,
	)
	return err
}

// GetMemberRequestIDs returns all request IDs belonging to a group.
func (s *GroupStore) GetMemberRequestIDs(ctx context.Context, groupID string) ([]string, error) {
	rows, err := s.DB.Query(ctx,
		`SELECT request_id FROM group_members WHERE group_id = $1`, groupID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// UpdateScore recalculates and persists the confidence_score after a member joins.
func (s *GroupStore) UpdateScore(ctx context.Context, groupID string, score float64) error {
	_, err := s.DB.Exec(ctx,
		`UPDATE ride_groups SET confidence_score = $1, updated_at = NOW() WHERE id = $2`,
		score, groupID,
	)
	return err
}

func scanGroups(rows interface {
	Next() bool
	Scan(...any) error
}) ([]models.RideGroup, error) {
	var groups []models.RideGroup
	for rows.Next() {
		var g models.RideGroup
		var expectedDep *time.Time
		if err := rows.Scan(
			&g.ID, &g.Status, &g.ConfidenceScore, &g.ArriveBy, &expectedDep,
			&g.DriverID, &g.DispatchAttempts, &g.TelegramMsgID, &g.CreatedAt, &g.UpdatedAt,
		); err != nil {
			return nil, err
		}
		g.ExpectedDeparture = expectedDep
		groups = append(groups, g)
	}
	return groups, nil
}
