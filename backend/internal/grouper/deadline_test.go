package grouper

import (
	"testing"
	"time"

	"github.com/Ksalgotra1/Marshal/internal/models"
)

func TestFastTrackShortRide(t *testing.T) {
	now := time.Now()
	// 5km ride. Ride min = (5/40)*60 = 7.5 mins.
	// Buffer = 10 mins. Total transit = 17.5 mins.
	// ArriveBy = now + 15 mins. Deadline = now + 15 - 17.5 = now - 2.5 mins
	// Deadline - now = -2.5 mins, which is <= 10 mins (fastTrackWindow).
	req := models.RideRequest{
		PickupLat:  0.0,
		PickupLng:  0.0,
		DropoffLat: 0.045, // roughly 5km
		DropoffLng: 0.0,
		PickupH3:   new(int64), DropoffH3: new(int64), ArriveBy: now.Add(15 * time.Minute),
	}

	fast, norm := splitByUrgency([]models.RideRequest{req})
	if len(fast) != 1 || len(norm) != 0 {
		t.Fatalf("expected 1 fast track, got %d fast, %d norm", len(fast), len(norm))
	}
}

func TestNormalLongWindow(t *testing.T) {
	now := time.Now()
	// 5km ride. Total transit = 17.5 mins.
	// ArriveBy = now + 2 hours. Deadline = now + 120 - 17.5 = now + 102.5 mins
	// Deadline - now = 102.5 mins, which is > 10 mins.
	req := models.RideRequest{
		PickupLat:  0.0,
		PickupLng:  0.0,
		DropoffLat: 0.045,
		DropoffLng: 0.0,
		PickupH3:   new(int64), DropoffH3: new(int64), ArriveBy: now.Add(2 * time.Hour),
	}

	fast, norm := splitByUrgency([]models.RideRequest{req})
	if len(fast) != 0 || len(norm) != 1 {
		t.Fatalf("expected 1 normal, got %d fast, %d norm", len(fast), len(norm))
	}
}

func TestFastTrackFormsAtTwo(t *testing.T) {
	now := time.Now()
	// 2 requests that should fast track
	r1 := models.RideRequest{ID: "r1", PickupLat: 0, PickupLng: 0, DropoffLat: 0.045, DropoffLng: 0, PickupH3: new(int64), DropoffH3: new(int64), ArriveBy: now.Add(15 * time.Minute)}
	r2 := models.RideRequest{ID: "r2", PickupLat: 0, PickupLng: 0, DropoffLat: 0.045, DropoffLng: 0, PickupH3: new(int64), DropoffH3: new(int64), ArriveBy: now.Add(15 * time.Minute)}

	fast, norm := splitByUrgency([]models.RideRequest{r1, r2})
	if len(fast) != 2 || len(norm) != 0 {
		t.Fatalf("expected 2 fast track, got %d fast", len(fast))
	}

	group := bestGroup(fast, map[string]bool{}, 2, 4)
	if len(group) != 2 {
		t.Fatalf("expected group of size 2, got %d", len(group))
	}
}

func TestNormalWaitsForBetter(t *testing.T) {
	now := time.Now()
	r1 := models.RideRequest{ID: "n1", PickupLat: 0, PickupLng: 0, DropoffLat: 0.045, DropoffLng: 0, PickupH3: new(int64), DropoffH3: new(int64), ArriveBy: now.Add(2 * time.Hour)}
	r2 := models.RideRequest{ID: "n2", PickupLat: 0, PickupLng: 0, DropoffLat: 0.045, DropoffLng: 0, PickupH3: new(int64), DropoffH3: new(int64), ArriveBy: now.Add(2 * time.Hour)}

	fast, norm := splitByUrgency([]models.RideRequest{r1, r2})
	if len(fast) != 0 || len(norm) != 2 {
		t.Fatalf("expected 2 normal, got %d normal", len(norm))
	}

	// It forms a group at size 2 (minSize=2, maxSize=4) -> "2 normal requests -> group forms, goes to rerank hook"
	group := bestGroup(norm, map[string]bool{}, 2, 4)
	if len(group) != 2 {
		t.Fatalf("expected group of size 2, got %d", len(group))
	}
}
