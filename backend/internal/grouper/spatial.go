package grouper

import (
	"math"

	"github.com/Ksalgotra1/Marshal/internal/geo"
	"github.com/Ksalgotra1/Marshal/internal/models"
)

type GroupType int

const (
	GroupTypeExact GroupType = iota
	GroupTypeEnRoute
)

type MatchResult struct {
	Type       GroupType
	Compatible bool
}

type Request = models.RideRequest

// CheckCorridor determines if two requests can share a route based on bearing
// and cross-track distance. It does NOT check pickup-cell distance (e.g. H3 GridDistance).
// The grid distance is handled entirely by the candidate pool selection (GridDisk)
// before this function is called. Checking it here again contradicts the 3-pass search
// and breaks en-route matching for distant pickups.
func CheckCorridor(a, b Request) MatchResult {
	// fail fast if H3 cells are missing
	if a.PickupH3 == nil || b.PickupH3 == nil {
		return MatchResult{Compatible: false}
	}

	// removed: pickup cells must be same or k-ring 1
	// (H3 distance check was redundant and pass-blind, breaking pass 2/3)

	pA := geo.LatLng{Lat: a.PickupLat, Lng: a.PickupLng}
	dA := geo.LatLng{Lat: a.DropoffLat, Lng: a.DropoffLng}
	pB := geo.LatLng{Lat: b.PickupLat, Lng: b.PickupLng}
	dB := geo.LatLng{Lat: b.DropoffLat, Lng: b.DropoffLng}

	bA := geo.BearingDeg(pA, dA)
	bB := geo.BearingDeg(pB, dB)

	// handle 0/360 bearing wrap
	diff := math.Abs(bA - bB)
	if diff > 180 {
		diff = 360 - diff
	}
	if diff >= 15 {
		return MatchResult{Compatible: false}
	}

	routeLenA := geo.HaversineKm(pA.Lat, pA.Lng, dA.Lat, dA.Lng)
	routeLenB := geo.HaversineKm(pB.Lat, pB.Lng, dB.Lat, dB.Lng)

	dropBToA := math.Abs(geo.CrossTrackDistanceKm(dB, pA, dA))
	dropAToB := math.Abs(geo.CrossTrackDistanceKm(dA, pB, dB))

	// dropoff must lie within α=0.15 * routeLen of the other's route
	if dropBToA > 0.15*routeLenA && dropAToB > 0.15*routeLenB {
		return MatchResult{Compatible: false}
	}

	// Note: With H3 resolution lowered to 7 for better pickup candidate reach,
	// dropoff cells are also coarser (~1.2km edge). This means dropoffs up to
	// ~1.2km apart may now be grouped as GroupTypeExact. This tradeoff is
	// acceptable as riders dropped within 1km plausibly count as the same trip.
	if a.DropoffH3 != nil && b.DropoffH3 != nil && *a.DropoffH3 == *b.DropoffH3 {
		return MatchResult{Type: GroupTypeExact, Compatible: true}
	}

	return MatchResult{Type: GroupTypeEnRoute, Compatible: true}
}
