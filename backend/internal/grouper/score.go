package grouper

import (
	"math"
	"time"

	"github.com/Ksalgotra1/Marshal/internal/geo"
)

type RequestMember struct {
	StudentID  string
	PickupLat  float64
	PickupLng  float64
	DropoffLat float64
	DropoffLng float64
	ArriveBy   time.Time
}

type RouteScoreInput struct {
	Members   []RequestMember
	GroupType GroupType
}

func meanPairwiseKm(points []geo.LatLng) float64 {
	if len(points) < 2 {
		return 0
	}
	var sum float64
	var count int
	for i := 0; i < len(points); i++ {
		for j := i + 1; j < len(points); j++ {
			sum += geo.HaversineKm(points[i].Lat, points[i].Lng, points[j].Lat, points[j].Lng)
			count++
		}
	}
	return sum / float64(count)
}

func stdDevKm(points []geo.LatLng) float64 {
	if len(points) < 2 {
		return 0
	}
	// compute centroid
	var latSum, lngSum float64
	for _, p := range points {
		latSum += p.Lat
		lngSum += p.Lng
	}
	n := float64(len(points))
	centroid := geo.LatLng{Lat: latSum / n, Lng: lngSum / n}

	var sumSq float64
	for _, p := range points {
		dist := geo.HaversineKm(p.Lat, p.Lng, centroid.Lat, centroid.Lng)
		sumSq += dist * dist
	}
	return math.Sqrt(sumSq / n)
}

func stdDevMinutes(times []time.Time) float64 {
	if len(times) < 2 {
		return 0
	}
	var sum float64
	for _, t := range times {
		sum += float64(t.Unix())
	}
	n := float64(len(times))
	meanSec := sum / n

	var sumSq float64
	for _, t := range times {
		diff := float64(t.Unix()) - meanSec
		sumSq += diff * diff
	}
	return math.Sqrt(sumSq/n) / 60.0
}

func minDropoffChainKm(lastPickup geo.LatLng, dropoffs []geo.LatLng) float64 {
	if len(dropoffs) == 0 {
		return 0
	}
	var permute func([]geo.LatLng, int)
	minDist := math.MaxFloat64

	permute = func(arr []geo.LatLng, i int) {
		if i == len(arr) {
			dist := geo.HaversineKm(lastPickup.Lat, lastPickup.Lng, arr[0].Lat, arr[0].Lng)
			for j := 0; j < len(arr)-1; j++ {
				dist += geo.HaversineKm(arr[j].Lat, arr[j].Lng, arr[j+1].Lat, arr[j+1].Lng)
			}
			if dist < minDist {
				minDist = dist
			}
			return
		}
		for j := i; j < len(arr); j++ {
			arr[i], arr[j] = arr[j], arr[i]
			permute(arr, i+1)
			arr[i], arr[j] = arr[j], arr[i]
		}
	}

	cpy := make([]geo.LatLng, len(dropoffs))
	copy(cpy, dropoffs)
	permute(cpy, 0)

	return minDist
}

func totalPickupChainKm(members []RequestMember) float64 {
	if len(members) == 0 {
		return 0
	}
	var total float64
	// sum pickup chain distances
	for i := 0; i < len(members)-1; i++ {
		total += geo.HaversineKm(members[i].PickupLat, members[i].PickupLng, members[i+1].PickupLat, members[i+1].PickupLng)
	}

	lastIdx := len(members) - 1
	lastPickup := geo.LatLng{Lat: members[lastIdx].PickupLat, Lng: members[lastIdx].PickupLng}

	var dropoffs []geo.LatLng
	for _, m := range members {
		dropoffs = append(dropoffs, geo.LatLng{Lat: m.DropoffLat, Lng: m.DropoffLng})
	}
	total += minDropoffChainKm(lastPickup, dropoffs)

	return total
}

func ComputeRouteScore(in RouteScoreInput) float64 {
	if len(in.Members) < 2 {
		return 0 // score expects >= 2 members
	}

	base := 10.0
	capacityBonuses := map[int]float64{2: 8.0, 3: 18.0, 4: 30.0}
	capacityBonus := capacityBonuses[len(in.Members)]

	directKm := geo.HaversineKm(in.Members[0].PickupLat, in.Members[0].PickupLng, in.Members[0].DropoffLat, in.Members[0].DropoffLng)
	detourKm := totalPickupChainKm(in.Members) - directKm
	if detourKm < 0 {
		detourKm = 0
	}

	var detourPenalty float64
	if directKm > 0 {
		detourPenalty = (detourKm / directKm) * 40.0
	} else {
		detourPenalty = detourKm * 40.0
	}

	var pickups []geo.LatLng
	for _, m := range in.Members {
		pickups = append(pickups, geo.LatLng{Lat: m.PickupLat, Lng: m.PickupLng})
	}
	meanPickupDist := meanPairwiseKm(pickups)

	densityBonus := 8.0
	if meanPickupDist > 0 {
		densityBonus = 5.0 / meanPickupDist
		if densityBonus > 8.0 {
			densityBonus = 8.0
		}
	}

	var dropoffs []geo.LatLng
	for _, m := range in.Members {
		dropoffs = append(dropoffs, geo.LatLng{Lat: m.DropoffLat, Lng: m.DropoffLng})
	}
	var spreadPenalty float64
	if in.GroupType != GroupTypeEnRoute {
		spreadPenalty = stdDevKm(dropoffs) * 10.0
	}

	var times []time.Time
	for _, m := range in.Members {
		times = append(times, m.ArriveBy)
	}
	timeBonus := 8.0 / (stdDevMinutes(times) + 1.0)
	if timeBonus > 8.0 {
		timeBonus = 8.0
	}

	var corridorPenalty float64
	if in.GroupType == GroupTypeEnRoute {
		corridorPenalty = detourKm * 6.0
	}

	return base + capacityBonus - detourPenalty + densityBonus - spreadPenalty + timeBonus - corridorPenalty
}
