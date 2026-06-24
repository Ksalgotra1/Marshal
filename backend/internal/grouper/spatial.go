package grouper

import (
	"math"

	"github.com/Ksalgotra1/Marshal/internal/geo"
	"github.com/Ksalgotra1/Marshal/internal/models"
	"github.com/uber/h3-go/v4"
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

func CheckCorridor(a, b Request) MatchResult {
	// fail fast if H3 cells are missing
	if a.PickupH3 == nil || b.PickupH3 == nil {
		return MatchResult{Compatible: false}
	}

	// pickup cells must be same or k-ring 1
	if *a.PickupH3 != *b.PickupH3 {
		dist, err := h3.GridDistance(h3.Cell(*a.PickupH3), h3.Cell(*b.PickupH3))
		if err != nil || dist > 1 {
			return MatchResult{Compatible: false}
		}
	}

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

	if a.DropoffH3 != nil && b.DropoffH3 != nil && *a.DropoffH3 == *b.DropoffH3 {
		return MatchResult{Type: GroupTypeExact, Compatible: true}
	}

	return MatchResult{Type: GroupTypeEnRoute, Compatible: true}
}
