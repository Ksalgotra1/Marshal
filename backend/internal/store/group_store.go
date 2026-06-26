package store

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/Ksalgotra1/Marshal/internal/models"
)

var ErrGroupNotJoinable = errors.New("group not joinable")

// GroupFilter holds query parameters for filtered group listing.
type GroupFilter struct {
	Status string // filter by status (empty = all)
	Limit  int    // max results (0 = default 20, max 100)
	Offset int    // result offset for pagination
}

// GroupStore handles ride group CRUD operations.
type GroupStore struct{ DB DBTX }

// Create inserts a new ride group and links its member requests.
// All operations run inside the caller's transaction.
func (s *GroupStore) Create(ctx context.Context, members []models.RideRequest, score float64, priority string) (string, error) {
	// Earliest arrive_by among members is the group's deadline
	arriveBy := members[0].ArriveBy
	for _, m := range members[1:] {
		if m.ArriveBy.Before(arriveBy) {
			arriveBy = m.ArriveBy
		}
	}

	var groupID string
	err := s.DB.QueryRow(ctx, `
		INSERT INTO ride_groups (priority, route_score, arrive_by)
		VALUES ($1, $2, $3)
		RETURNING id
	`, priority, score, arriveBy).Scan(&groupID)
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
		SELECT id, status, priority, route_score, arrive_by, expected_departure, driver_id,
		       dispatch_attempts, telegram_msg_id, created_at, updated_at
		FROM ride_groups
		WHERE status IN ('grouped', 'dispatching') AND driver_id IS NULL AND arrive_by > NOW()
		ORDER BY (CASE WHEN priority = 'high' THEN 0 ELSE 1 END), route_score DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("GroupStore.ListOpen: %w", err)
	}
	defer rows.Close()
	return scanGroups(rows)
}

// ListFiltered returns groups with optional status filter and limit.
func (s *GroupStore) ListFiltered(ctx context.Context, f GroupFilter) ([]models.RideGroup, error) {
	if f.Limit <= 0 || f.Limit > 100 {
		f.Limit = 20
	}
	if f.Offset < 0 {
		f.Offset = 0
	}

	query := `SELECT id, status, priority, route_score, arrive_by, expected_departure, driver_id,
	                 dispatch_attempts, telegram_msg_id, created_at, updated_at
	          FROM ride_groups`
	args := []any{}
	argIdx := 1

	if f.Status != "" {
		query += ` WHERE status = $` + strconv.Itoa(argIdx)
		args = append(args, f.Status)
		argIdx++
	}

	query += ` ORDER BY (CASE WHEN priority = 'high' THEN 0 ELSE 1 END), route_score DESC, created_at DESC LIMIT $` + strconv.Itoa(argIdx)
	args = append(args, f.Limit)
	argIdx++
	query += ` OFFSET $` + strconv.Itoa(argIdx)
	args = append(args, f.Offset)

	rows, err := s.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("GroupStore.ListFiltered: %w", err)
	}
	defer rows.Close()
	return scanGroups(rows)
}

// GetByID fetches a single group by ID.
func (s *GroupStore) GetByID(ctx context.Context, id string) (*models.RideGroup, error) {
	var g models.RideGroup
	err := s.DB.QueryRow(ctx, `
		SELECT id, status, priority, route_score, arrive_by, expected_departure, driver_id,
		       dispatch_attempts, telegram_msg_id, created_at, updated_at
		FROM ride_groups WHERE id = $1
	`, id).Scan(
		&g.ID, &g.Status, &g.Priority, &g.RouteScore, &g.ArriveBy, &g.ExpectedDeparture,
		&g.DriverID, &g.DispatchAttempts, &g.TelegramMsgID, &g.CreatedAt, &g.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("GroupStore.GetByID: %w", err)
	}
	return &g, nil
}

// GroupDetail is the enriched response for GET /api/groups/{id}.
type GroupDetail struct {
	Group   models.RideGroup     `json:"group"`
	Members []models.RideRequest `json:"members"`
}

// GetByIDWithMembers fetches a group and all its member ride requests in one call.
func (s *GroupStore) GetByIDWithMembers(ctx context.Context, id string) (*GroupDetail, error) {
	group, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	rows, err := s.DB.Query(ctx, `
		SELECT rr.id, rr.requester_name, rr.pickup_lat, rr.pickup_lng,
		       rr.dropoff_lat, rr.dropoff_lng, rr.pickup_h3, rr.dropoff_h3,
		       rr.arrive_by, rr.status, rr.created_at, rr.updated_at
		FROM ride_requests rr
		JOIN group_members gm ON rr.id = gm.request_id
		WHERE gm.group_id = $1
	`, id)
	if err != nil {
		return nil, fmt.Errorf("GroupStore.GetByIDWithMembers members: %w", err)
	}
	defer rows.Close()

	var members []models.RideRequest
	for rows.Next() {
		var r models.RideRequest
		if err := rows.Scan(
			&r.ID, &r.RequesterName, &r.PickupLat, &r.PickupLng,
			&r.DropoffLat, &r.DropoffLng, &r.PickupH3, &r.DropoffH3,
			&r.ArriveBy, &r.Status, &r.CreatedAt, &r.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("GroupStore.GetByIDWithMembers scan: %w", err)
		}
		members = append(members, r)
	}

	return &GroupDetail{Group: *group, Members: members}, nil
}

// PopHighestScore atomically grabs the highest-scoring unassigned group.
// This IS the priority queue pop — the partial B-tree index makes it O(log n).
// Returns nil if no groups are available.
func (s *GroupStore) PopHighestScore(ctx context.Context) (*models.RideGroup, error) {
	var g models.RideGroup
	err := s.DB.QueryRow(ctx, `
		SELECT id, status, priority, route_score, arrive_by, expected_departure, driver_id,
		       dispatch_attempts, telegram_msg_id, created_at, updated_at
		FROM ride_groups
		WHERE status = 'grouped' AND driver_id IS NULL
		ORDER BY (CASE WHEN priority = 'high' THEN 0 ELSE 1 END), route_score DESC
		LIMIT 1
		FOR UPDATE SKIP LOCKED
	`).Scan(
		&g.ID, &g.Status, &g.Priority, &g.RouteScore, &g.ArriveBy, &g.ExpectedDeparture,
		&g.DriverID, &g.DispatchAttempts, &g.TelegramMsgID, &g.CreatedAt, &g.UpdatedAt,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("GroupStore.PopHighestScore: %w", err)
	}
	return &g, nil
}

// AddMember joins a ride request to an existing open group.
func (s *GroupStore) AddMember(ctx context.Context, groupID, requestID string) error {
	cmd, err := s.DB.Exec(ctx, `
		UPDATE ride_groups SET updated_at = NOW()
		WHERE id = $1 AND status IN ('grouped', 'dispatching') AND driver_id IS NULL
	`, groupID)
	if err != nil {
		return fmt.Errorf("GroupStore.AddMember check: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrGroupNotJoinable
	}

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
	return nil
}

// GetActiveForDriver fetches the driver's currently assigned group.
func (s *GroupStore) GetActiveForDriver(ctx context.Context, driverID string) (*models.RideGroup, error) {
	var g models.RideGroup
	err := s.DB.QueryRow(ctx, `
		SELECT id, status, priority, route_score, arrive_by, expected_departure, driver_id,
		       dispatch_attempts, telegram_msg_id, created_at, updated_at
		FROM ride_groups
		WHERE driver_id = $1 AND status = 'assigned'
		LIMIT 1
	`, driverID).Scan(
		&g.ID, &g.Status, &g.Priority, &g.RouteScore, &g.ArriveBy, &g.ExpectedDeparture,
		&g.DriverID, &g.DispatchAttempts, &g.TelegramMsgID, &g.CreatedAt, &g.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("GroupStore.GetActiveForDriver: %w", err)
	}
	return &g, nil
}

// CompleteRide atomically marks a group and its members as completed using a CTE.
func (s *GroupStore) CompleteRide(ctx context.Context, groupID, driverID string) (bool, error) {
	var completed bool
	err := s.DB.QueryRow(ctx, `
		WITH completed AS (
			UPDATE ride_groups
			SET status = 'completed', completed_at = NOW(), updated_at = NOW()
			WHERE id = $1 AND driver_id = $2 AND status = 'assigned'
			RETURNING id
		),
		member_update AS (
			UPDATE ride_requests
			SET status = 'completed', updated_at = NOW()
			WHERE id IN (SELECT request_id FROM group_members WHERE group_id = $1)
			  AND EXISTS (SELECT 1 FROM completed)
			RETURNING id
		)
		SELECT EXISTS (SELECT 1 FROM completed)
	`, groupID, driverID).Scan(&completed)
	if err != nil {
		return false, fmt.Errorf("GroupStore.CompleteRide: %w", err)
	}
	return completed, nil
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

// UpdateScore recalculates and persists the route_score after a member joins.
func (s *GroupStore) UpdateScore(ctx context.Context, groupID string, score float64) error {
	_, err := s.DB.Exec(ctx,
		`UPDATE ride_groups SET route_score = $1, updated_at = NOW() WHERE id = $2`,
		score, groupID,
	)
	return err
}

// MarkDispatching transitions a group from 'grouped' to 'dispatching' and increments attempts.
// Called by the assigner after popping from the priority queue.
func (s *GroupStore) MarkDispatching(ctx context.Context, groupID string) error {
	_, err := s.DB.Exec(ctx, `
		UPDATE ride_groups
		SET status = 'dispatching', dispatch_attempts = dispatch_attempts + 1, updated_at = NOW()
		WHERE id = $1
	`, groupID)
	return err
}

// ClaimGroup atomically assigns a driver to a dispatching group.
// Returns rows affected: 1 = success, 0 = race lost (someone else claimed it).
// This is the concurrency-safe ACK — FOR UPDATE is implicit via the WHERE clause.
func (s *GroupStore) ClaimGroup(ctx context.Context, groupID, driverID string) (int64, error) {
	cmd, err := s.DB.Exec(ctx, `
		UPDATE ride_groups
		SET status = 'assigned', driver_id = $1, updated_at = NOW()
		WHERE id = $2 AND status = 'dispatching' AND driver_id IS NULL
	`, driverID, groupID)
	if err != nil {
		return 0, fmt.Errorf("GroupStore.ClaimGroup: %w", err)
	}
	return cmd.RowsAffected(), nil
}

// RevertTimedOut reverts groups stuck in 'dispatching' past their dispatch_timeout_at.
func (s *GroupStore) RevertTimedOut(ctx context.Context) ([]string, error) {
	rows, err := s.DB.Query(ctx, `
		UPDATE ride_groups
		SET status = 'grouped', updated_at = NOW()
		WHERE status = 'dispatching' AND dispatch_timeout_at < NOW()
		  AND dispatch_attempts < 3
		RETURNING id
	`)
	if err != nil {
		return nil, fmt.Errorf("GroupStore.RevertTimedOut: %w", err)
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

// CancelMaxRetries marks groups that have exceeded max dispatch attempts as 'cancelled'.
func (s *GroupStore) CancelMaxRetries(ctx context.Context) (int64, error) {
	cmd, err := s.DB.Exec(ctx, `
		UPDATE ride_groups
		SET status = 'cancelled', updated_at = NOW()
		WHERE status = 'dispatching' AND dispatch_attempts >= 3
	`)
	if err != nil {
		return 0, err
	}
	return cmd.RowsAffected(), nil
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
			&g.ID, &g.Status, &g.Priority, &g.RouteScore, &g.ArriveBy, &expectedDep,
			&g.DriverID, &g.DispatchAttempts, &g.TelegramMsgID, &g.CreatedAt, &g.UpdatedAt,
		); err != nil {
			return nil, err
		}
		g.ExpectedDeparture = expectedDep
		groups = append(groups, g)
	}
	return groups, nil
}

// SetTelegramMsgID updates the group's telegram_msg_id.
func (s *GroupStore) SetTelegramMsgID(ctx context.Context, groupID string, msgID int) error {
	_, err := s.DB.Exec(ctx, `UPDATE ride_groups SET telegram_msg_id = $1 WHERE id = $2`, msgID, groupID)
	return err
}

// SetDispatchTimeoutAt sets the group's dispatch_timeout_at.
func (s *GroupStore) SetDispatchTimeoutAt(ctx context.Context, groupID string, deadline time.Time) error {
	_, err := s.DB.Exec(ctx, `UPDATE ride_groups SET dispatch_timeout_at = $1 WHERE id = $2`, deadline, groupID)
	return err
}
