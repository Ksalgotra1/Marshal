package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Ksalgotra1/Marshal/internal/api"
	"github.com/Ksalgotra1/Marshal/internal/models"
	"github.com/stretchr/testify/suite"
)

type HTTPIntegrationSuite struct {
	suite.Suite
	handler http.Handler
}

func TestHTTPIntegrationSuite(t *testing.T) {
	suite.Run(t, new(HTTPIntegrationSuite))
}

func (s *HTTPIntegrationSuite) SetupTest() {
	h := &Handlers{}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/requests", h.HandleCreateRequest)
	mux.HandleFunc("POST /api/drivers", h.HandleRegisterDriver)
	s.handler = api.RequestIDMiddleware(api.CORSMiddleware(mux))
}

func (s *HTTPIntegrationSuite) TestCreateRequestInvalidJSONReturnsSafeStructuredError() {
	req := httptest.NewRequest(http.MethodPost, "/api/requests", bytes.NewBufferString("{bad-json"))
	rec := httptest.NewRecorder()

	s.handler.ServeHTTP(rec, req)

	s.Equal(http.StatusBadRequest, rec.Code)
	s.NotEmpty(rec.Header().Get("X-Request-ID"))

	var body map[string]string
	s.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &body))
	s.Equal("invalid JSON", body["error"])
	s.Equal(rec.Header().Get("X-Request-ID"), body["request_id"])
}

func (s *HTTPIntegrationSuite) TestCreateRequestValidationStopsBeforeDatabase() {
	payload := models.CreateRequestPayload{
		RequesterName: "Aman",
		PickupLat:     30.3545,
		PickupLng:     76.3658,
		DropoffLat:    30.7333,
		DropoffLng:    76.7794,
		ArriveBy:      time.Now().Add(2 * time.Minute),
	}
	body, err := json.Marshal(payload)
	s.Require().NoError(err)

	req := httptest.NewRequest(http.MethodPost, "/api/requests", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	s.handler.ServeHTTP(rec, req)

	s.Equal(http.StatusBadRequest, rec.Code)

	var response map[string]string
	s.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &response))
	s.Equal("arrive_by must be at least 10 minutes from now", response["error"])
	s.NotEmpty(response["request_id"])
}

func (s *HTTPIntegrationSuite) TestDriverRegistrationValidationStopsBeforeDatabase() {
	req := httptest.NewRequest(http.MethodPost, "/api/drivers", bytes.NewBufferString(`{"name":""}`))
	rec := httptest.NewRecorder()

	s.handler.ServeHTTP(rec, req)

	s.Equal(http.StatusBadRequest, rec.Code)

	var response map[string]string
	s.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &response))
	s.Equal("name and telegram_id are required", response["error"])
	s.NotEmpty(response["request_id"])
}

func (s *HTTPIntegrationSuite) TestCORSPreflightReturnsAllowedOriginAndRequestID() {
	req := httptest.NewRequest(http.MethodOptions, "/api/requests", nil)
	rec := httptest.NewRecorder()

	s.handler.ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code)
	s.Equal("http://localhost:5173", rec.Header().Get("Access-Control-Allow-Origin"))
	s.NotEmpty(rec.Header().Get("X-Request-ID"))
}
