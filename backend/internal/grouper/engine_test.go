package grouper

import (
	"context"
	"testing"
	"time"

	"github.com/Ksalgotra1/Marshal/internal/models"
	"github.com/uber/h3-go/v4"
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

func TestProcessPoolExcludesDistantPickups(t *testing.T) {
	// Two requests perfectly aligned in bearing, but ~30km apart in pickup.
	// H3 resolution 7 GridDisk(2) covers roughly 4-5km, so they will not enter the same pool.
	now := time.Now()
	
	// Chandigarh
	r1 := models.RideRequest{
		ID:         "1",
		PickupLat:  30.7333,
		PickupLng:  76.7794,
		DropoffLat: 28.7041, // Delhi
		DropoffLng: 77.1025,
		PickupH3:   ptrInt64(1), // Dummy H3, will be reassigned
		DropoffH3:  ptrInt64(2),
		ArriveBy:   now.Add(2 * time.Hour),
	}
	// Ambala (approx 40km away from Chandigarh)
	r2 := models.RideRequest{
		ID:         "2",
		PickupLat:  30.3782,
		PickupLng:  76.7767,
		DropoffLat: 28.7041, // Delhi
		DropoffLng: 77.1025,
		PickupH3:   ptrInt64(3), // Dummy H3, will be reassigned
		DropoffH3:  ptrInt64(2),
		ArriveBy:   now.Add(2 * time.Hour),
	}

	// Calculate realistic H3 cells for both at resolution 7
	c1, _ := h3.LatLngToCell(h3.LatLng{Lat: r1.PickupLat, Lng: r1.PickupLng}, 7)
	c2, _ := h3.LatLngToCell(h3.LatLng{Lat: r2.PickupLat, Lng: r2.PickupLng}, 7)
	
	hc1 := int64(c1)
	hc2 := int64(c2)
	r1.PickupH3 = &hc1
	r2.PickupH3 = &hc2

	e := &Engine{}
	groups := e.processPool(context.Background(), []models.RideRequest{r1, r2}, 2, 4)
	if len(groups) > 0 {
		t.Errorf("expected 0 groups due to processPool GridDisk exclusion of distant pickups, got %d", len(groups))
	}
}
