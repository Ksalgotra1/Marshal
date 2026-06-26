package store_test

import (
	"context"
	"testing"

	"github.com/Ksalgotra1/Marshal/internal/store"
	"github.com/Ksalgotra1/Marshal/internal/testdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDriverPresence_MarkStaleOffline(t *testing.T) {
	db := testdb.Start(t)
	ds := &store.DriverStore{DB: db.Pool}
	ctx := context.Background()

	// 1. Online, seen recently -> Should stay online
	// 2. Online, seen long ago -> Should flip to offline
	// 3. Busy, seen long ago -> Should stay busy
	// 4. Online, never seen (NULL) -> Should flip to offline
	
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO drivers (name, telegram_id, status, last_seen_at) VALUES 
		('recent_online', 1, 'online', NOW()),
		('stale_online', 2, 'online', NOW() - INTERVAL '30 minutes'),
		('stale_busy', 3, 'busy', NOW() - INTERVAL '30 minutes'),
		('null_online', 4, 'online', NULL)
	`)
	require.NoError(t, err)

	affected, err := ds.MarkStaleOffline(ctx, 15)
	require.NoError(t, err)
	assert.Equal(t, int64(2), affected) // stale_online and null_online flipped

	// Verify statuses
	d1, _ := ds.GetByTelegramID(ctx, 1)
	assert.Equal(t, "online", d1.Status)

	d2, _ := ds.GetByTelegramID(ctx, 2)
	assert.Equal(t, "offline", d2.Status)

	d3, _ := ds.GetByTelegramID(ctx, 3)
	assert.Equal(t, "busy", d3.Status)

	d4, _ := ds.GetByTelegramID(ctx, 4)
	assert.Equal(t, "offline", d4.Status)
}

func TestDriverPresence_CountOnline(t *testing.T) {
	db := testdb.Start(t)
	ds := &store.DriverStore{DB: db.Pool}
	ctx := context.Background()

	_, err := db.Pool.Exec(ctx, `
		INSERT INTO drivers (name, telegram_id, status, last_seen_at) VALUES 
		('recent_online_1', 1, 'online', NOW()),
		('recent_online_2', 2, 'online', NOW() - INTERVAL '10 minutes'),
		('stale_online', 3, 'online', NOW() - INTERVAL '20 minutes'),
		('null_online', 4, 'online', NULL),
		('recent_busy', 5, 'busy', NOW()),
		('recent_offline', 6, 'offline', NOW())
	`)
	require.NoError(t, err)

	count, err := ds.CountOnline(ctx, 15)
	require.NoError(t, err)
	// Only recent_online_1 and recent_online_2 should be counted
	assert.Equal(t, 2, count)
}
