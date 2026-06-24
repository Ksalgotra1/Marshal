package grouper

import (
	"time"

	"github.com/Ksalgotra1/Marshal/internal/geo"
	"github.com/Ksalgotra1/Marshal/internal/models"
)

const (
	avgCitySpeedKmh = 40.0
	bufferMinutes   = 10
	fastTrackWindow = 10 * time.Minute
)

// computeFormationDeadline calculates the deadline for a group to form.
func computeFormationDeadline(req models.RideRequest) time.Time {
	rideKm := geo.HaversineKm(req.PickupLat, req.PickupLng, req.DropoffLat, req.DropoffLng)
	rideMin := (rideKm / avgCitySpeedKmh) * 60
	return req.ArriveBy.Add(-time.Duration(rideMin) * time.Minute).Add(-bufferMinutes * time.Minute)
}

// splitByUrgency splits requests into fast-track and normal pools based on deadline.
func splitByUrgency(reqs []models.RideRequest) (fastTrack, normal []models.RideRequest) {
	now := time.Now()
	for _, r := range reqs {
		deadline := computeFormationDeadline(r)
		if deadline.Sub(now) <= fastTrackWindow {
			fastTrack = append(fastTrack, r)
		} else {
			normal = append(normal, r)
		}
	}
	return
}
