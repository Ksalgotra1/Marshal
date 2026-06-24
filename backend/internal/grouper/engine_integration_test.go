package grouper

import (
	"context"
	"testing"
	"time"

	"github.com/Ksalgotra1/Marshal/internal/models"
	"github.com/Ksalgotra1/Marshal/internal/realtime"
	"github.com/Ksalgotra1/Marshal/internal/store"
	"github.com/Ksalgotra1/Marshal/internal/testdb"
	"github.com/stretchr/testify/suite"
)

type GrouperIntegrationSuite struct {
	suite.Suite
	db     *testdb.Instance
	ctx    context.Context
	events *recordingPublisher
}

func TestGrouperIntegrationSuite(t *testing.T) {
	suite.Run(t, new(GrouperIntegrationSuite))
}

func (s *GrouperIntegrationSuite) SetupSuite() {
	s.ctx = context.Background()
	s.db = testdb.Start(s.T())
}

func (s *GrouperIntegrationSuite) SetupTest() {
	testdb.Truncate(s.ctx, s.T(), s.db.Pool)
	s.events = &recordingPublisher{}
}

func (s *GrouperIntegrationSuite) TestRunFormsCompatibleGroupAndBroadcasts() {
	rs := &store.RequestStore{DB: s.db.Pool}
	for i, name := range []string{"Aman", "Bani", "Chirag"} {
		_, err := rs.Create(s.ctx, &models.CreateRequestPayload{
			RequesterName: name,
			PickupLat:     30.3545 + float64(i)*0.0001,
			PickupLng:     76.3658 + float64(i)*0.0001,
			DropoffLat:    30.7333 + float64(i)*0.0001,
			DropoffLng:    76.7794 + float64(i)*0.0001,
			ArriveBy:      time.Now().Add(45*time.Minute + time.Duration(i)*5*time.Minute),
		})
		s.Require().NoError(err)
	}

	(&Engine{Pool: s.db.Pool, Events: s.events}).Run(s.ctx)

	var groupedRequests int
	s.Require().NoError(s.db.Pool.QueryRow(s.ctx, `SELECT COUNT(*) FROM ride_requests WHERE status = 'grouped'`).Scan(&groupedRequests))
	s.Equal(3, groupedRequests)

	var groupID string
	var status string
	var memberCount int
	s.Require().NoError(s.db.Pool.QueryRow(s.ctx, `
		SELECT rg.id, rg.status, COUNT(gm.request_id)
		FROM ride_groups rg
		JOIN group_members gm ON rg.id = gm.group_id
		GROUP BY rg.id, rg.status
	`).Scan(&groupID, &status, &memberCount))
	s.Equal("grouped", status)
	s.Equal(3, memberCount)

	s.Require().Len(s.events.events, 1)
	s.Equal("group:formed", s.events.events[0].Type)
	s.Equal(groupID, s.events.events[0].GroupID)
	s.Contains(s.events.rooms[0], "global")
	s.Contains(s.events.rooms[0], groupID)
}

func (s *GrouperIntegrationSuite) TestRunDoesNotGroupWhenUnderTargetSize() {
	rs := &store.RequestStore{DB: s.db.Pool}
	for _, name := range []string{"Aman"} {
		_, err := rs.Create(s.ctx, &models.CreateRequestPayload{
			RequesterName: name,
			PickupLat:     30.3545,
			PickupLng:     76.3658,
			DropoffLat:    30.7333,
			DropoffLng:    76.7794,
			ArriveBy:      time.Now().Add(45 * time.Minute),
		})
		s.Require().NoError(err)
	}

	(&Engine{Pool: s.db.Pool, Events: s.events}).Run(s.ctx)

	var groups int
	s.Require().NoError(s.db.Pool.QueryRow(s.ctx, `SELECT COUNT(*) FROM ride_groups`).Scan(&groups))
	s.Zero(groups)
	s.Empty(s.events.events)
}

func (s *GrouperIntegrationSuite) TestRunConcurrentLockContention() {
	rs := &store.RequestStore{DB: s.db.Pool}

	// Create 6 requests (exactly enough for 2 groups of 3)
	for i := 0; i < 6; i++ {
		_, err := rs.Create(s.ctx, &models.CreateRequestPayload{
			RequesterName: "Rider",
			PickupLat:     30.3545,
			PickupLng:     76.3658,
			DropoffLat:    30.7333,
			DropoffLng:    76.7794,
			ArriveBy:      time.Now().Add(45 * time.Minute),
		})
		s.Require().NoError(err)
	}

	// Run two engines concurrently to simulate race conditions on pending requests
	done := make(chan struct{})

	go func() {
		(&Engine{Pool: s.db.Pool, Events: s.events}).Run(s.ctx)
		done <- struct{}{}
	}()

	go func() {
		(&Engine{Pool: s.db.Pool, Events: s.events}).Run(s.ctx)
		done <- struct{}{}
	}()

	<-done
	<-done

	// Verify exactly 2 groups were formed, no requests were duplicated or missed
	var groups int
	s.Require().NoError(s.db.Pool.QueryRow(s.ctx, `SELECT COUNT(*) FROM ride_groups`).Scan(&groups))
	s.Equal(2, groups, "Should form exactly 2 groups without lock contention dropping or duplicating requests")
}

func (s *GrouperIntegrationSuite) TestRunIgnoresStaleRequests() {
	rs := &store.RequestStore{DB: s.db.Pool}

	// Create 2 requests, but one is already past its arrive_by time
	for i := 0; i < 2; i++ {
		arriveBy := time.Now().Add(45 * time.Minute)
		if i == 0 {
			arriveBy = time.Now().Add(-10 * time.Minute) // Stale
		}

		_, err := rs.Create(s.ctx, &models.CreateRequestPayload{
			RequesterName: "Rider",
			PickupLat:     30.3545,
			PickupLng:     76.3658,
			DropoffLat:    30.7333,
			DropoffLng:    76.7794,
			ArriveBy:      arriveBy,
		})
		s.Require().NoError(err)
	}

	(&Engine{Pool: s.db.Pool, Events: s.events}).Run(s.ctx)

	// The stale request should prevent the other 2 from forming a group of 3
	var groups int
	s.Require().NoError(s.db.Pool.QueryRow(s.ctx, `SELECT COUNT(*) FROM ride_groups`).Scan(&groups))
	s.Equal(0, groups, "Should not form a group if one of the required members is stale")
}

type recordingPublisher struct {
	rooms  [][]string
	events []realtime.Event
}

func (p *recordingPublisher) BroadcastMulti(rooms []string, event realtime.Event) {
	p.rooms = append(p.rooms, rooms)
	p.events = append(p.events, event)
}
