package store

import (
	"context"
	"fmt"

	"github.com/Ksalgotra1/Marshal/internal/models"
	"github.com/uber/h3-go/v4"
)

// RequestStore handles ride request CRUD operations.
type RequestStore struct{ DB DBTX }

// Create inserts a new ride request, computing H3 cells server-side.
func (s *RequestStore) Create(ctx context.Context, req *models.CreateRequestPayload) (string, error) {
	// Compute H3 hexagonal indices at resolution 9 (~174m radius)
	pickupCell, err := h3.LatLngToCell(h3.LatLng{Lat: req.PickupLat, Lng: req.PickupLng}, 9)
	if err != nil {
		return "", fmt.Errorf("invalid pickup coordinates for H3: %w", err)
	}
	dropoffCell, err := h3.LatLngToCell(h3.LatLng{Lat: req.DropoffLat, Lng: req.DropoffLng}, 9)
	if err != nil {
		return "", fmt.Errorf("invalid dropoff coordinates for H3: %w", err)
	}

	query := `
		INSERT INTO ride_requests (requester_name, pickup_lat, pickup_lng, dropoff_lat, dropoff_lng, arrive_by, pickup_h3, dropoff_h3)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`
	var id string
	err = s.DB.QueryRow(ctx, query,
		req.RequesterName, req.PickupLat, req.PickupLng,
		req.DropoffLat, req.DropoffLng, req.ArriveBy,
		int64(pickupCell), int64(dropoffCell),
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("CreateRideRequest: %w", err)
	}
	return id, nil
}

// GetByID fetches a single ride request by ID, including its group_id if grouped.
func (s *RequestStore) GetByID(ctx context.Context, id string) (*models.RideRequest, error) {
	query := `SELECT rr.id, rr.requester_name, rr.pickup_lat, rr.pickup_lng, rr.dropoff_lat, rr.dropoff_lng,
	                 rr.pickup_h3, rr.dropoff_h3, rr.arrive_by, rr.status, gm.group_id,
	                 rr.created_at, rr.updated_at
	          FROM ride_requests rr
	          LEFT JOIN group_members gm ON rr.id = gm.request_id
	          WHERE rr.id = $1`
	var r models.RideRequest
	err := s.DB.QueryRow(ctx, query, id).Scan(
		&r.ID, &r.RequesterName, &r.PickupLat, &r.PickupLng,
		&r.DropoffLat, &r.DropoffLng, &r.PickupH3, &r.DropoffH3,
		&r.ArriveBy, &r.Status, &r.GroupID,
		&r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("GetRequestByID: %w", err)
	}
	return &r, nil
}

// RequestFilter holds query parameters for filtered request listing.
type RequestFilter struct {
	Status string
	Limit  int
	Offset int
}

// ListFiltered returns requests with optional status filter and limit.
func (s *RequestStore) ListFiltered(ctx context.Context, f RequestFilter) ([]models.RideRequest, error) {
	if f.Limit <= 0 || f.Limit > 100 {
		f.Limit = 20
	}
	if f.Offset < 0 {
		f.Offset = 0
	}

	query := `SELECT rr.id, rr.requester_name, rr.pickup_lat, rr.pickup_lng, rr.dropoff_lat, rr.dropoff_lng,
	                 rr.pickup_h3, rr.dropoff_h3, rr.arrive_by, rr.status, gm.group_id,
	                 rr.created_at, rr.updated_at
	          FROM ride_requests rr
	          LEFT JOIN group_members gm ON rr.id = gm.request_id`
	args := []any{}
	argIdx := 1

	if f.Status != "" {
		query += fmt.Sprintf(` WHERE rr.status = $%d`, argIdx)
		args = append(args, f.Status)
		argIdx++
	}

	query += fmt.Sprintf(` ORDER BY rr.created_at DESC LIMIT $%d`, argIdx)
	args = append(args, f.Limit)
	argIdx++
	query += fmt.Sprintf(` OFFSET $%d`, argIdx)
	args = append(args, f.Offset)

	rows, err := s.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("RequestStore.ListFiltered: %w", err)
	}
	defer rows.Close()

	var requests []models.RideRequest
	for rows.Next() {
		var r models.RideRequest
		if err := rows.Scan(
			&r.ID, &r.RequesterName, &r.PickupLat, &r.PickupLng,
			&r.DropoffLat, &r.DropoffLng, &r.PickupH3, &r.DropoffH3,
			&r.ArriveBy, &r.Status, &r.GroupID,
			&r.CreatedAt, &r.UpdatedAt,
		); err == nil {
			requests = append(requests, r)
		}
	}
	return requests, nil
}

// FetchPendingLocked grabs all pending requests with FOR UPDATE SKIP LOCKED.
// This is the grouper's intake query — no other worker can touch these rows.
func (s *RequestStore) FetchPendingLocked(ctx context.Context) ([]models.RideRequest, error) {
	query := `
		SELECT id, requester_name, pickup_lat, pickup_lng, dropoff_lat, dropoff_lng,
		       pickup_h3, dropoff_h3, arrive_by, status, created_at, updated_at
			FROM ride_requests
			WHERE status = 'pending'
			ORDER BY created_at ASC
			FOR UPDATE SKIP LOCKED
		`
	rows, err := s.DB.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("FetchPendingLocked: %w", err)
	}
	defer rows.Close()

	var requests []models.RideRequest
	for rows.Next() {
		var r models.RideRequest
		if err := rows.Scan(
			&r.ID, &r.RequesterName, &r.PickupLat, &r.PickupLng,
			&r.DropoffLat, &r.DropoffLng, &r.PickupH3, &r.DropoffH3,
			&r.ArriveBy, &r.Status, &r.CreatedAt, &r.UpdatedAt,
		); err == nil {
			requests = append(requests, r)
		}
	}
	return requests, nil
}

