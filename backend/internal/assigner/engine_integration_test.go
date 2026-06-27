package assigner

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Ksalgotra1/Marshal/internal/models"
	"github.com/Ksalgotra1/Marshal/internal/realtime"
	"github.com/Ksalgotra1/Marshal/internal/store"
	"github.com/Ksalgotra1/Marshal/internal/testdb"
	"github.com/stretchr/testify/suite"
)

type AssignerIntegrationSuite struct {
	suite.Suite
	db     *testdb.Instance
	ctx    context.Context
	events *recordingPublisher
}

func TestAssignerIntegrationSuite(t *testing.T) {
	suite.Run(t, new(AssignerIntegrationSuite))
}

func (s *AssignerIntegrationSuite) SetupSuite() {
	s.ctx = context.Background()
	s.db = testdb.Start(s.T())
}

func (s *AssignerIntegrationSuite) SetupTest() {
	testdb.Truncate(s.ctx, s.T(), s.db.Pool)
	s.events = &recordingPublisher{}

	// Add an online driver so standard dispatch tests proceed
	var driverID string
	err := s.db.Pool.QueryRow(s.ctx, `
		INSERT INTO drivers (name, telegram_id, status, last_seen_at) VALUES ('Test Driver', 12345, 'online', NOW()) RETURNING id
	`).Scan(&driverID)
	s.Require().NoError(err)
}

func (s *AssignerIntegrationSuite) TestRunDispatchesGroupedPriorityQueueAndBroadcasts() {
	lowID := s.createGroup(10)
	highID := s.createGroup(95)

	Run(s.ctx, s.db.Pool, s.events, nil, 15)

	statuses := s.groupStatuses()
	s.Equal("dispatching", statuses[lowID])
	s.Equal("dispatching", statuses[highID])
	s.Len(s.events.events, 2)
	s.Equal("group:dispatching", s.events.events[0].Type)
	s.Equal(highID, s.events.events[0].GroupID)
	s.Equal("group:dispatching", s.events.events[1].Type)
	s.Equal(lowID, s.events.events[1].GroupID)
}

func (s *AssignerIntegrationSuite) TestRunCancelsMaxRetriesAndRedispatchesTimedOutGroups() {
	cancelID := s.insertRawGroup("dispatching", 3, time.Now().Add(time.Hour), time.Now().Add(-10*time.Minute), 50)
	timedOutID := s.insertRawGroup("dispatching", 1, time.Now().Add(time.Hour), time.Now().Add(-10*time.Minute), 60)

	Run(s.ctx, s.db.Pool, s.events, nil, 15)

	statuses := s.groupStatuses()
	s.Equal("cancelled", statuses[cancelID])
	s.Equal("dispatching", statuses[timedOutID])

	var attempts int
	s.Require().NoError(s.db.Pool.QueryRow(s.ctx, `SELECT dispatch_attempts FROM ride_groups WHERE id = $1`, timedOutID).Scan(&attempts))
	s.Equal(2, attempts)
}

func (s *AssignerIntegrationSuite) TestAssignerPrioritizesFastTrackOverHigherScore() {
	normalID := s.insertRawGroupPriority("grouped", 0, time.Now().Add(time.Hour), time.Now(), 90, models.PriorityNormal)
	fastID := s.insertRawGroupPriority("grouped", 0, time.Now().Add(time.Hour), time.Now(), 20, models.PriorityHigh)

	Run(s.ctx, s.db.Pool, s.events, nil, 15)

	s.Len(s.events.events, 2)
	s.Equal("group:dispatching", s.events.events[0].Type)
	s.Equal(fastID, s.events.events[0].GroupID) // Fast gets dispatched FIRST

	s.Equal("group:dispatching", s.events.events[1].Type)
	s.Equal(normalID, s.events.events[1].GroupID)
}

func (s *AssignerIntegrationSuite) TestRunConcurrentLockContention() {
	// Create exactly 5 groups
	for i := 0; i < 5; i++ {
		s.createGroup(float64(i * 10))
	}

	// Verify we have 5 grouped
	var initialCount int
	s.Require().NoError(s.db.Pool.QueryRow(s.ctx, `SELECT COUNT(*) FROM ride_groups WHERE status = 'grouped'`).Scan(&initialCount))
	s.Equal(5, initialCount)

	// Run two assigner loops concurrently
	done := make(chan struct{})
	go func() {
		Run(s.ctx, s.db.Pool, s.events, nil, 15)
		done <- struct{}{}
	}()
	go func() {
		Run(s.ctx, s.db.Pool, s.events, nil, 15)
		done <- struct{}{}
	}()

	<-done
	<-done

	// Verify all 5 were popped and marked dispatching exactly once
	// If lock contention failed, some might still be 'grouped' or one worker might have crashed.
	var dispatchingCount int
	s.Require().NoError(s.db.Pool.QueryRow(s.ctx, `SELECT COUNT(*) FROM ride_groups WHERE status = 'dispatching'`).Scan(&dispatchingCount))
	s.Equal(5, dispatchingCount)
}

func (s *AssignerIntegrationSuite) TestAssignerSkipsCycleWhenNoDriversOnline() {
	// Set driver offline
	_, err := s.db.Pool.Exec(s.ctx, `UPDATE drivers SET status = 'offline'`)
	s.Require().NoError(err)

	s.createGroup(10) // Create grouped ride

	Run(s.ctx, s.db.Pool, s.events, nil, 15) // No drivers online

	s.Empty(s.events.events) // Nothing should have been dispatched
}

func (s *AssignerIntegrationSuite) createGroup(score float64) string {
	rs := &store.RequestStore{DB: s.db.Pool}
	var members []models.RideRequest
	for i := 0; i < 3; i++ {
		id, err := rs.Create(s.ctx, &models.CreateRequestPayload{
			RequesterName: "Rider",
			PickupLat:     30.3545 + float64(i)*0.0001,
			PickupLng:     76.3658 + float64(i)*0.0001,
			DropoffLat:    30.7333 + float64(i)*0.0001,
			DropoffLng:    76.7794 + float64(i)*0.0001,
			ArriveBy:      time.Now().Add(time.Hour),
		})
		s.Require().NoError(err)
		req, err := rs.GetByID(s.ctx, id)
		s.Require().NoError(err)
		members = append(members, *req)
	}
	id, err := (&store.GroupStore{DB: s.db.Pool}).Create(s.ctx, members, score, models.PriorityNormal)
	s.Require().NoError(err)
	_, _ = s.db.Pool.Exec(s.ctx, `UPDATE ride_groups SET created_at = NOW() - interval '5 minutes' WHERE id = $1`, id)
	return id
}

func (s *AssignerIntegrationSuite) insertRawGroup(status string, attempts int, arriveBy, updatedAt time.Time, score float64) string {
	var id string
	s.Require().NoError(s.db.Pool.QueryRow(s.ctx, `
		INSERT INTO ride_groups (status, dispatch_attempts, arrive_by, updated_at, dispatch_timeout_at, route_score)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, status, attempts, arriveBy, updatedAt, updatedAt, score).Scan(&id))
	_, _ = s.db.Pool.Exec(s.ctx, `UPDATE ride_groups SET created_at = NOW() - interval '5 minutes' WHERE id = $1`, id)
	return id
}

func (s *AssignerIntegrationSuite) insertRawGroupPriority(status string, attempts int, arriveBy, updatedAt time.Time, score float64, priority string) string {
	var id string
	s.Require().NoError(s.db.Pool.QueryRow(s.ctx, `
		INSERT INTO ride_groups (status, dispatch_attempts, arrive_by, updated_at, dispatch_timeout_at, route_score, priority)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, status, attempts, arriveBy, updatedAt, updatedAt, score, priority).Scan(&id))
	_, _ = s.db.Pool.Exec(s.ctx, `UPDATE ride_groups SET created_at = NOW() - interval '5 minutes' WHERE id = $1`, id)
	return id
}

func (s *AssignerIntegrationSuite) groupStatuses() map[string]string {
	rows, err := s.db.Pool.Query(s.ctx, `SELECT id, status FROM ride_groups`)
	s.Require().NoError(err)
	defer rows.Close()

	statuses := map[string]string{}
	for rows.Next() {
		var id string
		var status string
		s.Require().NoError(rows.Scan(&id, &status))
		statuses[id] = status
	}
	return statuses
}

type recordingPublisher struct {
	mu     sync.Mutex
	rooms  [][]string
	events []realtime.Event
}

func (p *recordingPublisher) BroadcastMulti(rooms []string, event realtime.Event) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.rooms = append(p.rooms, rooms)
	p.events = append(p.events, event)
}
