package grouper

import (
	"testing"
	"time"

	"github.com/Ksalgotra1/Marshal/internal/models"
)

func ptrInt64(i int64) *int64 { return &i }

func TestGroupGetsRouteScore(t *testing.T) {
	now := time.Now()
	r1 := models.RideRequest{
		ID:         "1",
		PickupLat:  30.3398,
		PickupLng:  76.3869,
		DropoffLat: 30.7333,
		DropoffLng: 76.7794,
		PickupH3:   ptrInt64(1),
		DropoffH3:  ptrInt64(2),
		ArriveBy:   now,
	}
	r2 := models.RideRequest{
		ID:         "2",
		PickupLat:  30.3398,
		PickupLng:  76.3869,
		DropoffLat: 30.7333,
		DropoffLng: 76.7794,
		PickupH3:   ptrInt64(1),
		DropoffH3:  ptrInt64(2),
		ArriveBy:   now,
	}

	group := []models.RideRequest{r1, r2}
	score := computeRouteScoreForGroup(group)
	if score == 0 {
		t.Errorf("expected score != 0")
	}

	res := CheckCorridor(r1, r2)
	manualScore := ComputeRouteScore(RouteScoreInput{
		Members: []RequestMember{
			{StudentID: r1.ID, PickupLat: r1.PickupLat, PickupLng: r1.PickupLng, DropoffLat: r1.DropoffLat, DropoffLng: r1.DropoffLng, ArriveBy: r1.ArriveBy},
			{StudentID: r2.ID, PickupLat: r2.PickupLat, PickupLng: r2.PickupLng, DropoffLat: r2.DropoffLat, DropoffLng: r2.DropoffLng, ArriveBy: r2.ArriveBy},
		},
		GroupType: res.Type,
	})

	if score != manualScore {
		t.Errorf("expected engine computed score %f to match manual score %f", score, manualScore)
	}
}
