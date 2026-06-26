package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/Ksalgotra1/Marshal/internal/store"
	"github.com/Ksalgotra1/Marshal/internal/testdb"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompleteRide(t *testing.T) {
	db := testdb.Start(t)
	gs := &store.GroupStore{DB: db.Pool}

	ctx := context.Background()

	// Setup: insert driver, request, and group
	var driverID string
	err := db.Pool.QueryRow(ctx, "INSERT INTO drivers (name, telegram_id, status) VALUES ('test_driver', 12345, 'busy') RETURNING id").Scan(&driverID)
	require.NoError(t, err)

	var reqID string
	err = db.Pool.QueryRow(ctx, "INSERT INTO ride_requests (requester_name, pickup_lat, pickup_lng, dropoff_lat, dropoff_lng, arrive_by, status) VALUES ('req', 0, 0, 0, 0, NOW(), 'grouped') RETURNING id").Scan(&reqID)
	require.NoError(t, err)

	var groupID string
	err = db.Pool.QueryRow(ctx, "INSERT INTO ride_groups (status, driver_id, arrive_by) VALUES ('assigned', $1, NOW()) RETURNING id", driverID).Scan(&groupID)
	require.NoError(t, err)

	_, err = db.Pool.Exec(ctx, "INSERT INTO group_members (request_id, group_id) VALUES ($1, $2)", reqID, groupID)
	require.NoError(t, err)

	t.Run("correct driver and group", func(t *testing.T) {
		completed, err := gs.CompleteRide(ctx, groupID, driverID)
		require.NoError(t, err)
		assert.True(t, completed)

		// Verify ride_groups
		var groupStatus string
		var completedAt *time.Time
		err = db.Pool.QueryRow(ctx, "SELECT status, completed_at FROM ride_groups WHERE id = $1", groupID).Scan(&groupStatus, &completedAt)
		require.NoError(t, err)
		assert.Equal(t, "completed", groupStatus)
		assert.NotNil(t, completedAt)

		// Verify ride_requests
		var reqStatus string
		err = db.Pool.QueryRow(ctx, "SELECT status FROM ride_requests WHERE id = $1", reqID).Scan(&reqStatus)
		require.NoError(t, err)
		assert.Equal(t, "completed", reqStatus)

		// GetActiveForDriver returns nil
		active, err := gs.GetActiveForDriver(ctx, driverID)
		require.Error(t, err)
		require.ErrorIs(t, err, pgx.ErrNoRows)
		assert.Nil(t, active)
	})

	t.Run("wrong driver", func(t *testing.T) {
		// Insert another group
		var g2 string
		err = db.Pool.QueryRow(ctx, "INSERT INTO ride_groups (status, driver_id, arrive_by) VALUES ('assigned', $1, NOW()) RETURNING id", driverID).Scan(&g2)
		require.NoError(t, err)
		
		var r2 string
		err = db.Pool.QueryRow(ctx, "INSERT INTO ride_requests (requester_name, pickup_lat, pickup_lng, dropoff_lat, dropoff_lng, arrive_by, status) VALUES ('req2', 0, 0, 0, 0, NOW(), 'grouped') RETURNING id").Scan(&r2)
		require.NoError(t, err)
		_, err = db.Pool.Exec(ctx, "INSERT INTO group_members (request_id, group_id) VALUES ($1, $2)", r2, g2)
		require.NoError(t, err)

		// Attempt CompleteRide with a wrong driver UUID
		completed, err := gs.CompleteRide(ctx, g2, "11111111-1111-1111-1111-111111111111")
		require.NoError(t, err)
		assert.False(t, completed)

		// Verify it wasn't touched
		var groupStatus string
		err = db.Pool.QueryRow(ctx, "SELECT status FROM ride_groups WHERE id = $1", g2).Scan(&groupStatus)
		require.NoError(t, err)
		assert.Equal(t, "assigned", groupStatus)

		var reqStatus string
		err = db.Pool.QueryRow(ctx, "SELECT status FROM ride_requests WHERE id = $1", r2).Scan(&reqStatus)
		require.NoError(t, err)
		assert.Equal(t, "grouped", reqStatus)
	})
}
