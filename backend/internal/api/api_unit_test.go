package api_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Ksalgotra1/Marshal/internal/api"
	"github.com/stretchr/testify/suite"
)

type ErrorResponseUnitSuite struct {
	suite.Suite
}

func TestErrorResponseUnitSuite(t *testing.T) {
	suite.Run(t, new(ErrorResponseUnitSuite))
}

func (s *ErrorResponseUnitSuite) TestWriteRequestErrorIncludesRequestIDAndHidesInternalError() {
	handler := api.RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		api.WriteRequestError(w, r, http.StatusInternalServerError, "safe public message", errors.New("password=db-secret"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	s.Equal(http.StatusInternalServerError, rec.Code)
	s.NotEmpty(rec.Header().Get("X-Request-ID"))

	var body map[string]string
	s.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &body))
	s.Equal("safe public message", body["error"])
	s.Equal(rec.Header().Get("X-Request-ID"), body["request_id"])
	s.NotContains(rec.Body.String(), "db-secret")
}

func (s *ErrorResponseUnitSuite) TestRequestIDReturnsEmptyWithoutMiddleware() {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	s.Empty(api.RequestID(req))
}
