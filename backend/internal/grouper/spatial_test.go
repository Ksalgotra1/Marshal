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
