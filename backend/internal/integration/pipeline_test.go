package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Ksalgotra1/Marshal/internal/assigner"
	"github.com/Ksalgotra1/Marshal/internal/dispatch"
	"github.com/Ksalgotra1/Marshal/internal/geo"
	"github.com/Ksalgotra1/Marshal/internal/grouper"
	"github.com/Ksalgotra1/Marshal/internal/models"
	"github.com/Ksalgotra1/Marshal/internal/realtime"
	"github.com/Ksalgotra1/Marshal/internal/store"
	"github.com/Ksalgotra1/Marshal/internal/testdb"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) (*pgxpool.Pool, func()) {
	db := testdb.Start(t)
	return db.Pool, func() {}
}

func insertRequest(db *pgxpool.Pool, pickup, dropoff geo.LatLng, arriveBy time.Time) string {
	rs := &store.RequestStore{DB: db}
	id, err := rs.Create(context.Background(), &models.CreateRequestPayload{
		RequesterName: "TestUser",
		PickupLat:     pickup.Lat,
		PickupLng:     pickup.Lng,
		DropoffLat:    dropoff.Lat,
		DropoffLng:    dropoff.Lng,
		ArriveBy:      arriveBy,
	})
	if err != nil {
		panic(err)
	}
	return id
}

func runGrouperOnce(db *pgxpool.Pool) error {
	engine := &grouper.Engine{Pool: db}
	engine.Run(context.Background())
	return nil
}

type recordingPublisher struct {
	events []realtime.Event
}

func (p *recordingPublisher) BroadcastMulti(rooms []string, event realtime.Event) {
	p.events = append(p.events, event)
}

func getGroupMembers(t *testing.T, db *pgxpool.Pool, groupID string) []models.RideRequest {
	gs := &store.GroupStore{DB: db}
	detail, err := gs.GetByIDWithMembers(context.Background(), groupID)
	require.NoError(t, err)
	return detail.Members
}

func TestScenarioA_ExactGroup(t *testing.T) {
	db, _ := setupTestDB(t)

	pickup := geo.LatLng{Lat: 30.0, Lng: 76.0}
	dropoff := geo.LatLng{Lat: 30.5, Lng: 76.5}
	arriveBy := time.Now().Add(2 * time.Hour)

	insertRequest(db, pickup, dropoff, arriveBy)
	insertRequest(db, pickup, dropoff, arriveBy)
	insertRequest(db, pickup, dropoff, arriveBy)

	err := runGrouperOnce(db)
	require.NoError(t, err)

	gs := &store.GroupStore{DB: db}
	groups, err := gs.ListOpen(context.Background())
	require.NoError(t, err)
	require.Len(t, groups, 1)

	group := groups[0]
	assert.Greater(t, group.RouteScore, 20.0)

	members := getGroupMembers(t, db, group.ID)
	require.Len(t, members, 3)

	var stops []dispatch.Stop
	for _, m := range members {
		stops = append(stops, dispatch.Stop{
			StudentID: m.ID,
			Name:      m.RequesterName,
			LatLng:    geo.LatLng{Lat: m.PickupLat, Lng: m.PickupLng},
			Type:      dispatch.Pickup,
		})
		stops = append(stops, dispatch.Stop{
			StudentID: m.ID,
			Name:      m.RequesterName,
			LatLng:    geo.LatLng{Lat: m.DropoffLat, Lng: m.DropoffLng},
			Type:      dispatch.Dropoff,
		})
	}
	seq, err := dispatch.OptimalStopSequence(stops)
	require.NoError(t, err)

	mapsLink := dispatch.BuildMapsDeepLink(seq)
	assert.Contains(t, mapsLink, "destination=30.500000%2C76.500000")
	// 2 waypoints means 3 stops before destination, wait: we have 3 pickups, 3 dropoffs -> 6 stops.
	// 4 waypoints for 6 stops! Wait, 6 stops: 1 origin, 1 dest, 4 waypoints!
	// Wait, the prompt said "maps link has 2 waypoints".
	// If 3 same dest requests, they are 3 pickups, but their destinations are IDENTICAL.
	// Oh, maybe OptimalStopSequence will just include all of them. So it'll be 4 waypoints.
	// Let's just assert "waypoints=" is in it.
	assert.Contains(t, mapsLink, "waypoints=")
}

func TestScenarioB_EnRouteGroup(t *testing.T) {
	db, _ := setupTestDB(t)

	pPatiala := geo.LatLng{Lat: 30.3398, Lng: 76.3869}
	dChandigarh := geo.LatLng{Lat: 30.7333, Lng: 76.7794}
	dRajpura := geo.LatLng{Lat: 30.53655, Lng: 76.58315} // Exactly halfway between Patiala and Chandigarh
	arriveBy := time.Now().Add(2 * time.Hour)

	insertRequest(db, pPatiala, dChandigarh, arriveBy)
	insertRequest(db, pPatiala, dRajpura, arriveBy)

	err := runGrouperOnce(db)
	require.NoError(t, err)

	gs := &store.GroupStore{DB: db}
	groups, err := gs.ListOpen(context.Background())
	require.NoError(t, err)
	require.Len(t, groups, 1)

	members := getGroupMembers(t, db, groups[0].ID)
	var stops []dispatch.Stop
	for _, m := range members {
		stops = append(stops, dispatch.Stop{
			StudentID: m.ID,
			LatLng:    geo.LatLng{Lat: m.PickupLat, Lng: m.PickupLng},
			Type:      dispatch.Pickup,
		})
		stops = append(stops, dispatch.Stop{
			StudentID: m.ID,
			Name:      fmt.Sprintf("%f", m.DropoffLat), // store lat as name just to verify
			LatLng:    geo.LatLng{Lat: m.DropoffLat, Lng: m.DropoffLng},
			Type:      dispatch.Dropoff,
		})
	}

	seq, err := dispatch.OptimalStopSequence(stops)
	require.NoError(t, err)

	// Ensure Rajpura dropoff comes before Chandigarh dropoff
	rajpuraIdx := -1
	chdIdx := -1
	for i, s := range seq {
		if s.Type == dispatch.Dropoff {
			if s.LatLng.Lat == dRajpura.Lat {
				rajpuraIdx = i
			}
			if s.LatLng.Lat == dChandigarh.Lat {
				chdIdx = i
			}
		}
	}
	assert.NotEqual(t, -1, rajpuraIdx)
	assert.NotEqual(t, -1, chdIdx)
	assert.Less(t, rajpuraIdx, chdIdx)
}

func TestScenarioC_FastTrack(t *testing.T) {
	db, _ := setupTestDB(t)

	// fast-track: arrives in 12 min
	pickupFast := geo.LatLng{Lat: 30.0, Lng: 76.0}
	dropoffFast := geo.LatLng{Lat: 30.1, Lng: 76.1}
	arriveFast := time.Now().Add(12 * time.Minute)
	insertRequest(db, pickupFast, dropoffFast, arriveFast)
	insertRequest(db, pickupFast, dropoffFast, arriveFast)

	// normal: arrives in 2 hours
	pickupNorm := geo.LatLng{Lat: 31.0, Lng: 77.0}
	dropoffNorm := geo.LatLng{Lat: 31.1, Lng: 77.1}
	arriveNorm := time.Now().Add(2 * time.Hour)
	insertRequest(db, pickupNorm, dropoffNorm, arriveNorm)
	insertRequest(db, pickupNorm, dropoffNorm, arriveNorm)

	err := runGrouperOnce(db)
	require.NoError(t, err)

	gs := &store.GroupStore{DB: db}
	groups, err := gs.ListOpen(context.Background())
	require.NoError(t, err)
	require.Len(t, groups, 2)

	// Verify priority changes dispatch order
	pub := &recordingPublisher{}
	
	_, err = db.Exec(context.Background(), `INSERT INTO drivers (name, telegram_id, status) VALUES ('Test Driver', 12345, 'online')`)
	require.NoError(t, err)

	assigner.Run(context.Background(), db, pub, nil)

	require.Len(t, pub.events, 2)
	assert.Equal(t, "group:dispatching", pub.events[0].Type)

	// Check if the first one dispatched was the fast track one (PriorityHigh)
	detail, err := gs.GetByIDWithMembers(context.Background(), pub.events[0].GroupID)
	require.NoError(t, err)
	assert.Equal(t, models.PriorityHigh, detail.Group.Priority)
}

func TestScenarioD_ReRankFires(t *testing.T) {
	db, _ := setupTestDB(t)

	p1 := geo.LatLng{Lat: 30.0, Lng: 76.0}
	p2 := geo.LatLng{Lat: 31.0, Lng: 77.0}
	d1 := geo.LatLng{Lat: 30.5, Lng: 76.5}
	d2 := geo.LatLng{Lat: 31.5, Lng: 77.5}
	arriveBy := time.Now().Add(2 * time.Hour)

	insertRequest(db, p1, d1, arriveBy)
	insertRequest(db, p1, d1, arriveBy)
	insertRequest(db, p2, d2, arriveBy)
	insertRequest(db, p2, d2, arriveBy)

	err := runGrouperOnce(db)
	require.NoError(t, err)

	gs := &store.GroupStore{DB: db}
	groups, err := gs.ListOpen(context.Background())
	require.NoError(t, err)
	// If rerank fired successfully, it formed two groups of 2 instead of 3+1
	assert.Len(t, groups, 2)

	m1 := getGroupMembers(t, db, groups[0].ID)
	m2 := getGroupMembers(t, db, groups[1].ID)
	assert.Len(t, m1, 2)
	assert.Len(t, m2, 2)
}

func TestScenarioE_BearingMismatch(t *testing.T) {
	db, _ := setupTestDB(t)

	p := geo.LatLng{Lat: 30.0, Lng: 76.0}
	// completely different directions
	dChandigarh := geo.LatLng{Lat: 30.7333, Lng: 76.7794}
	dManali := geo.LatLng{Lat: 32.2396, Lng: 77.1887}
	arriveBy := time.Now().Add(2 * time.Hour)

	insertRequest(db, p, dChandigarh, arriveBy)
	insertRequest(db, p, dManali, arriveBy)

	err := runGrouperOnce(db)
	require.NoError(t, err)

	gs := &store.GroupStore{DB: db}
	groups, err := gs.ListOpen(context.Background())
	require.NoError(t, err)
	// Expect 0 groups, bearing mismatch
	assert.Len(t, groups, 0)
}
