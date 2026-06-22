//go:build integration

package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Ksalgotra1/Marshal/internal/api"
	"github.com/Ksalgotra1/Marshal/internal/models"
	"github.com/Ksalgotra1/Marshal/internal/testdb"
	"github.com/stretchr/testify/suite"
)

type HandlersIntegrationSuite struct {
	suite.Suite
	db      *testdb.Instance
	ctx     context.Context
	handler http.Handler
	handlers *Handlers
}

func TestHandlersIntegrationSuite(t *testing.T) {
	suite.Run(t, new(HandlersIntegrationSuite))
}

func (s *HandlersIntegrationSuite) SetupSuite() {
	s.ctx = context.Background()
	s.db = testdb.Start(s.T())
	
	s.handlers = &Handlers{Pool: s.db.Pool}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/requests", s.handlers.HandleCreateRequest)
	mux.HandleFunc("POST /api/groups/{id}/claim", s.handlers.HandleClaimGroup)
	mux.HandleFunc("POST /api/groups/{id}/join", s.handlers.HandleJoinGroup)
	
	s.handler = api.RequestIDMiddleware(api.CORSMiddleware(mux))
}

func (s *HandlersIntegrationSuite) SetupTest() {
	testdb.Truncate(s.ctx, s.T(), s.db.Pool)
}

func (s *HandlersIntegrationSuite) TestCreateRequest_TimeTravelRejection() {
	// Attempt to create a request 2 hours in the past
	payload := models.CreateRequestPayload{
		RequesterName: "Marty",
		PickupLat:     30.3545,
		PickupLng:     76.3658,
		DropoffLat:    30.7333,
		DropoffLng:    76.7794,
		ArriveBy:      time.Now().Add(-2 * time.Hour), // Time travel!
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/requests", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	s.handler.ServeHTTP(rec, req)

	s.Equal(http.StatusBadRequest, rec.Code)
	
	var response map[string]string
	json.Unmarshal(rec.Body.Bytes(), &response)
	s.Contains(response["error"], "at least 10 minutes from now")
}

func (s *HandlersIntegrationSuite) TestClaimGroup_ConcurrentConflict() {
	// Create a dummy driver
	var driverID string
	s.Require().NoError(s.db.Pool.QueryRow(s.ctx, `
		INSERT INTO drivers (name, telegram_id, status) VALUES ('TestDriver', 12345, 'online') RETURNING id
	`).Scan(&driverID))
	
	// Create a second driver
	var driver2ID string
	s.Require().NoError(s.db.Pool.QueryRow(s.ctx, `
		INSERT INTO drivers (name, telegram_id, status) VALUES ('TestDriver2', 67890, 'online') RETURNING id
	`).Scan(&driver2ID))

	// Create a dispatching group
	var groupID string
	s.Require().NoError(s.db.Pool.QueryRow(s.ctx, `
		INSERT INTO ride_groups (status, confidence_score, arrive_by) VALUES ('dispatching', 50, NOW() + INTERVAL '1 hour') RETURNING id
	`).Scan(&groupID))

	done := make(chan int)
	
	claimFunc := func(dID string) {
		payload := map[string]string{"driver_id": dID}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/api/groups/"+groupID+"/claim", bytes.NewReader(body))
		// We must set path values since we bypass the actual server mux router parameters in test without chi/gorilla.
		// Actually, standard Go 1.22 mux handles it, but httptest.NewRequest doesn't automatically parse path values if not routed through the actual server.
		// Wait, we pass it to `s.handler.ServeHTTP`, which includes the Mux, so it should parse the path correctly.
		rec := httptest.NewRecorder()
		s.handler.ServeHTTP(rec, req)
		done <- rec.Code
	}

	// Fire claims simultaneously
	go claimFunc(driverID)
	go claimFunc(driver2ID)

	code1 := <-done
	code2 := <-done

	// One should succeed (200), one should fail (409 Conflict)
	if code1 == http.StatusOK {
		s.Equal(http.StatusConflict, code2)
	} else {
		s.Equal(http.StatusConflict, code1)
		s.Equal(http.StatusOK, code2)
	}
}

func (s *HandlersIntegrationSuite) TestJoinGroup_CapacityLimit() {
	// Create a group
	var groupID string
	s.Require().NoError(s.db.Pool.QueryRow(s.ctx, `
		INSERT INTO ride_groups (status, confidence_score, arrive_by) VALUES ('grouped', 50, NOW() + INTERVAL '1 hour') RETURNING id
	`).Scan(&groupID))

	// Add 4 members to max it out
	for i := 0; i < 4; i++ {
		var reqID string
		s.Require().NoError(s.db.Pool.QueryRow(s.ctx, `
			INSERT INTO ride_requests (requester_name, arrive_by, status, pickup_lat, pickup_lng, dropoff_lat, dropoff_lng) VALUES ('Rider', NOW() + INTERVAL '1 hour', 'grouped', 0, 0, 0, 0) RETURNING id
		`).Scan(&reqID))
		s.Require().NoError(s.db.Pool.QueryRow(s.ctx, `
			INSERT INTO group_members (group_id, request_id) VALUES ($1, $2) RETURNING request_id
		`, groupID, reqID).Scan(&reqID))
	}

	// Try to add a 5th member
	var newReqID string
	s.Require().NoError(s.db.Pool.QueryRow(s.ctx, `
		INSERT INTO ride_requests (requester_name, arrive_by, status, pickup_lat, pickup_lng, dropoff_lat, dropoff_lng) VALUES ('Rider5', NOW() + INTERVAL '1 hour', 'pending', 0, 0, 0, 0) RETURNING id
	`).Scan(&newReqID))

	payload := map[string]string{"request_id": newReqID}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/groups/"+groupID+"/join", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	s.handler.ServeHTTP(rec, req)

	s.Equal(http.StatusBadRequest, rec.Code)
	var response map[string]string
	json.Unmarshal(rec.Body.Bytes(), &response)
	s.Contains(response["error"], "group is full")
}
