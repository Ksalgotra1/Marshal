package grouper

import (
	"testing"
	"time"

	"github.com/Ksalgotra1/Marshal/internal/models"
	"github.com/stretchr/testify/suite"
)

type GrouperUnitSuite struct {
	suite.Suite
	now time.Time
}

func TestGrouperUnitSuite(t *testing.T) {
	suite.Run(t, new(GrouperUnitSuite))
}

func (s *GrouperUnitSuite) SetupTest() {
	s.now = time.Now()
}

func (s *GrouperUnitSuite) TestBestGroupPicksCompatibleTripleAndIgnoresAssigned() {
	pool := []models.RideRequest{
		s.request("skip", 30.3545, 76.3658, 30.7333, 76.7794, s.now.Add(45*time.Minute)),
		s.request("a", 30.3545, 76.3658, 30.7333, 76.7794, s.now.Add(45*time.Minute)),
		s.request("b", 30.3547, 76.3660, 30.7334, 76.7796, s.now.Add(50*time.Minute)),
		s.request("c", 30.3548, 76.3661, 30.7332, 76.7795, s.now.Add(55*time.Minute)),
		s.request("far", 30.3548, 76.3661, 31.6339, 74.8723, s.now.Add(55*time.Minute)),
	}

	group := bestGroup(pool, map[string]bool{"skip": true})

	s.Require().Len(group, 3)
	s.ElementsMatch([]string{"a", "b", "c"}, []string{group[0].ID, group[1].ID, group[2].ID})
}

func (s *GrouperUnitSuite) TestCompatibilityRejectsDistantDropoffOrWideTimeWindow() {
	a := s.request("a", 30.3545, 76.3658, 30.7333, 76.7794, s.now.Add(45*time.Minute))
	b := s.request("b", 30.3547, 76.3660, 30.7334, 76.7796, s.now.Add(50*time.Minute))
	farDropoff := s.request("far", 30.3548, 76.3661, 31.6339, 74.8723, s.now.Add(55*time.Minute))
	late := s.request("late", 30.3548, 76.3661, 30.7332, 76.7795, s.now.Add(4*time.Hour))

	s.True(groupCompatible([]models.RideRequest{a, b}))
	s.False(groupCompatible([]models.RideRequest{a, farDropoff}))
	s.False(groupCompatible([]models.RideRequest{a, late}))
}


func (s *GrouperUnitSuite) request(id string, pickupLat, pickupLng, dropoffLat, dropoffLng float64, arriveBy time.Time) models.RideRequest {
	return models.RideRequest{
		ID:         id,
		PickupLat:  pickupLat,
		PickupLng:  pickupLng,
		DropoffLat: dropoffLat,
		DropoffLng: dropoffLng,
		ArriveBy:   arriveBy,
		CreatedAt:  s.now.Add(-10 * time.Minute),
	}
}
