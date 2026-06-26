package grouper

import (
	"testing"

	"github.com/Ksalgotra1/Marshal/internal/models"
)

func ptr(i int64) *int64 { return &i }

func TestExactMatch(t *testing.T) {
	reqA := models.RideRequest{
		PickupLat: 30.3398, PickupLng: 76.3869,
		DropoffLat: 30.7333, DropoffLng: 76.7794,
		PickupH3: ptr(1), DropoffH3: ptr(2),
	}
	reqB := models.RideRequest{
		PickupLat: 30.3398, PickupLng: 76.3869,
		DropoffLat: 30.7333, DropoffLng: 76.7794,
		PickupH3: ptr(1), DropoffH3: ptr(2),
	}

	res := CheckCorridor(reqA, reqB)
	if !res.Compatible {
		t.Errorf("expected compatible, got false")
	}
	if res.Type != GroupTypeExact {
		t.Errorf("expected GroupTypeExact, got %v", res.Type)
	}
}

func TestEnRouteMatch(t *testing.T) {
	reqA := models.RideRequest{
		PickupLat: 30.3398, PickupLng: 76.3869,
		DropoffLat: 30.7333, DropoffLng: 76.7794,
		PickupH3: ptr(1), DropoffH3: ptr(2),
	}
	reqB := models.RideRequest{
		PickupLat: 30.3412, PickupLng: 76.3901,
		DropoffLat: 30.4833, DropoffLng: 76.5833,
		PickupH3: ptr(1), DropoffH3: ptr(3),
	}

	res := CheckCorridor(reqA, reqB)
	if !res.Compatible {
		t.Errorf("expected compatible, got false")
	}
	if res.Type != GroupTypeEnRoute {
		t.Errorf("expected GroupTypeEnRoute, got %v", res.Type)
	}
}

func TestBearingMismatch(t *testing.T) {
	reqA := models.RideRequest{
		PickupLat: 30.3398, PickupLng: 76.3869,
		DropoffLat: 30.7333, DropoffLng: 76.7794,
		PickupH3: ptr(1), DropoffH3: ptr(2),
	}
	reqB := models.RideRequest{
		PickupLat: 30.3398, PickupLng: 76.3869,
		DropoffLat: 31.0, DropoffLng: 76.0,
		PickupH3: ptr(1), DropoffH3: ptr(4),
	}

	res := CheckCorridor(reqA, reqB)
	if res.Compatible {
		t.Errorf("expected incompatible, got true")
	}
}

func TestPickupTooFar(t *testing.T) {
	reqA := models.RideRequest{
		PickupLat: 30.3398, PickupLng: 76.3869,
		DropoffLat: 30.7333, DropoffLng: 76.7794,
		PickupH3: ptr(1), DropoffH3: ptr(2),
	}
	reqB := models.RideRequest{
		PickupLat: 30.7333, PickupLng: 76.7794,
		DropoffLat: 28.7041, DropoffLng: 77.1025,
		PickupH3: ptr(5), DropoffH3: ptr(6),
	}

	res := CheckCorridor(reqA, reqB)
	if res.Compatible {
		t.Errorf("expected incompatible due to pickup far, got true")
	}
}

func TestPickupDistanceInCorridor(t *testing.T) {
	// Sector 17 (30.7414, 76.7680) to Delhi (28.7041, 77.1025)
	reqA := models.RideRequest{
		PickupLat: 30.7414, PickupLng: 76.7680,
		DropoffLat: 28.7041, DropoffLng: 77.1025,
		PickupH3: ptr(1), DropoffH3: ptr(2),
	}
	// Sector 43 (30.7241, 76.7323) to Delhi - roughly 3.8km away from Sector 17
	reqB := models.RideRequest{
		PickupLat: 30.7241, PickupLng: 76.7323,
		DropoffLat: 28.7041, DropoffLng: 77.1025,
		PickupH3: ptr(3), DropoffH3: ptr(2),
	}

	// This should now be compatible because the H3 grid distance check was removed.
	res := CheckCorridor(reqA, reqB)
	if !res.Compatible {
		t.Errorf("expected compatible for ~3.8km pickup with matching corridor, got false")
	}
}

func TestPickupDistanceInCorridorBearingMismatch(t *testing.T) {
	// Sector 17 (30.7414, 76.7680) to Delhi (28.7041, 77.1025)
	reqA := models.RideRequest{
		PickupLat: 30.7414, PickupLng: 76.7680,
		DropoffLat: 28.7041, DropoffLng: 77.1025,
		PickupH3: ptr(1), DropoffH3: ptr(2),
	}
	// Sector 43 (30.7241, 76.7323) to Ludhiana (30.9010, 75.8573) - completely different direction
	reqB := models.RideRequest{
		PickupLat: 30.7241, PickupLng: 76.7323,
		DropoffLat: 30.9010, DropoffLng: 75.8573,
		PickupH3: ptr(3), DropoffH3: ptr(4),
	}

	res := CheckCorridor(reqA, reqB)
	if res.Compatible {
		t.Errorf("expected incompatible due to bearing mismatch, got true")
	}
}

func TestPickupDistanceInCorridorNonIdenticalDropoff(t *testing.T) {
	// Sector 17 (30.7414, 76.7680) to Delhi Airport (28.5562, 77.1000)
	reqA := models.RideRequest{
		PickupLat: 30.7414, PickupLng: 76.7680,
		DropoffLat: 28.5562, DropoffLng: 77.1000,
		PickupH3: ptr(1), DropoffH3: ptr(2),
	}
	// Sector 43 (30.7241, 76.7323) to New Delhi Railway Station (28.6433, 77.2197)
	// Dropoffs are ~13km apart, but they share the vast majority of the 250km route (corridor).
	reqB := models.RideRequest{
		PickupLat: 30.7241, PickupLng: 76.7323,
		DropoffLat: 28.6433, DropoffLng: 77.2197,
		PickupH3: ptr(3), DropoffH3: ptr(4),
	}

	res := CheckCorridor(reqA, reqB)
	if !res.Compatible {
		t.Errorf("expected compatible for non-identical but close dropoffs, got false")
	}
}
