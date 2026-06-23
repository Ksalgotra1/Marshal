package grouper

import (
	"math"

	"github.com/Ksalgotra1/Marshal/internal/models"
	"github.com/uber/h3-go/v4"
)

type LatLng struct {
	Lat float64
	Lng float64
}

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

func haversineKm(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 6371.0
	dLat := (lat2 - lat1) * math.Pi / 180.0
	dLng := (lng2 - lng1) * math.Pi / 180.0
	lat1Rad := lat1 * math.Pi / 180.0
	lat2Rad := lat2 * math.Pi / 180.0

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Sin(dLng/2)*math.Sin(dLng/2)*math.Cos(lat1Rad)*math.Cos(lat2Rad)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

func bearingDeg(from, to LatLng) float64 {
	lat1 := from.Lat * math.Pi / 180.0
	lat2 := to.Lat * math.Pi / 180.0
	dLng := (to.Lng - from.Lng) * math.Pi / 180.0

	y := math.Sin(dLng) * math.Cos(lat2)
	x := math.Cos(lat1)*math.Sin(lat2) - math.Sin(lat1)*math.Cos(lat2)*math.Cos(dLng)

	b := math.Atan2(y, x) * 180.0 / math.Pi
	if b < 0 {
		b += 360.0
	}
	return b
}

func crossTrackDistanceKm(point, lineStart, lineEnd LatLng) float64 {
	d13 := haversineKm(lineStart.Lat, lineStart.Lng, point.Lat, point.Lng)
	theta13 := bearingDeg(lineStart, point) * math.Pi / 180.0
	theta12 := bearingDeg(lineStart, lineEnd) * math.Pi / 180.0

	const R = 6371.0
	return math.Asin(math.Sin(d13/R)*math.Sin(theta13-theta12)) * R
}

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

	pA := LatLng{a.PickupLat, a.PickupLng}
	dA := LatLng{a.DropoffLat, a.DropoffLng}
	pB := LatLng{b.PickupLat, b.PickupLng}
	dB := LatLng{b.DropoffLat, b.DropoffLng}

	bA := bearingDeg(pA, dA)
	bB := bearingDeg(pB, dB)

	// handle 0/360 bearing wrap
	diff := math.Abs(bA - bB)
	if diff > 180 {
		diff = 360 - diff
	}
	if diff >= 25 {
		return MatchResult{Compatible: false}
	}

	routeLenA := haversineKm(pA.Lat, pA.Lng, dA.Lat, dA.Lng)
	routeLenB := haversineKm(pB.Lat, pB.Lng, dB.Lat, dB.Lng)

	dropBToA := math.Abs(crossTrackDistanceKm(dB, pA, dA))
	dropAToB := math.Abs(crossTrackDistanceKm(dA, pB, dB))

	// dropoff must lie within α=0.15 * routeLen of the other's route
	if dropBToA > 0.15*routeLenA && dropAToB > 0.15*routeLenB {
		return MatchResult{Compatible: false}
	}

	if a.DropoffH3 != nil && b.DropoffH3 != nil && *a.DropoffH3 == *b.DropoffH3 {
		return MatchResult{Type: GroupTypeExact, Compatible: true}
	}

	return MatchResult{Type: GroupTypeEnRoute, Compatible: true}
}
