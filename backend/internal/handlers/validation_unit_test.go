package handlers

import (
	"testing"
	"time"

	"github.com/Ksalgotra1/Marshal/internal/models"
	"github.com/stretchr/testify/suite"
)

type ValidationUnitSuite struct {
	suite.Suite
}

func TestValidationUnitSuite(t *testing.T) {
	suite.Run(t, new(ValidationUnitSuite))
}

func (s *ValidationUnitSuite) TestValidatePayloadAcceptsValidTrip() {
	payload := models.CreateRequestPayload{
		RequesterName: "Aman",
		PickupLat:     30.3545,
		PickupLng:     76.3658,
		DropoffLat:    30.7333,
		DropoffLng:    76.7794,
		ArriveBy:      time.Now().Add(45 * time.Minute),
	}

	s.NoError(validatePayload(&payload))
}

func (s *ValidationUnitSuite) TestValidatePayloadRejectsUnsafeOrInvalidCoordinates() {
	cases := []struct {
		name    string
		mutate  func(*models.CreateRequestPayload)
		message string
	}{
		{
			name:    "missing requester",
			mutate:  func(p *models.CreateRequestPayload) { p.RequesterName = "" },
			message: "requester_name is required",
		},
		{
			name:    "pickup latitude out of range",
			mutate:  func(p *models.CreateRequestPayload) { p.PickupLat = 91 },
			message: "pickup_lat out of range",
		},
		{
			name:    "same pickup and dropoff",
			mutate:  func(p *models.CreateRequestPayload) { p.DropoffLat, p.DropoffLng = p.PickupLat, p.PickupLng },
			message: "pickup and dropoff cannot be the same location",
		},
		{
			name:    "arrive by too soon",
			mutate:  func(p *models.CreateRequestPayload) { p.ArriveBy = time.Now().Add(2 * time.Minute) },
			message: "arrive_by must be at least 10 minutes from now",
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			payload := models.CreateRequestPayload{
				RequesterName: "Aman",
				PickupLat:     30.3545,
				PickupLng:     76.3658,
				DropoffLat:    30.7333,
				DropoffLng:    76.7794,
				ArriveBy:      time.Now().Add(45 * time.Minute),
			}
			tc.mutate(&payload)

			err := validatePayload(&payload)
			s.Require().Error(err)
			s.Equal(tc.message, err.Error())
		})
	}
}
