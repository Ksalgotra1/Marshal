package dispatch

import (
	"fmt"
	"math"
	"net/url"
	"strings"

	"github.com/Ksalgotra1/Marshal/internal/geo"
	"github.com/Ksalgotra1/Marshal/internal/models"
)

type StopType int

const (
	Pickup StopType = iota
	Dropoff
)

type Stop struct {
	StudentID string
	Name      string
	Location  string
	LatLng    geo.LatLng
	Type      StopType
}

func OptimalStopSequence(stops []Stop) ([]Stop, error) {
	if len(stops) > 8 {
		return nil, fmt.Errorf("too many stops for optimal routing")
	}

	var bestSeq []Stop
	var minCost = math.MaxFloat64

	var permute func(arr []Stop, n int)
	permute = func(arr []Stop, n int) {
		if n == 1 {
			if isValid(arr) {
				cost := routeCost(arr)
				if cost < minCost {
					minCost = cost
					bestSeq = make([]Stop, len(arr))
					copy(bestSeq, arr)
				}
			}
			return
		}
		for i := 0; i < n; i++ {
			permute(arr, n-1)
			if n%2 == 1 {
				arr[0], arr[n-1] = arr[n-1], arr[0]
			} else {
				arr[i], arr[n-1] = arr[n-1], arr[i]
			}
		}
	}

	cpy := make([]Stop, len(stops))
	copy(cpy, stops)
	permute(cpy, len(cpy))

	if bestSeq == nil {
		return nil, fmt.Errorf("no valid sequence found")
	}
	return bestSeq, nil
}

func isValid(seq []Stop) bool {
	pickedUp := make(map[string]bool)
	for _, s := range seq {
		if s.Type == Pickup {
			pickedUp[s.StudentID] = true
		} else if s.Type == Dropoff {
			if !pickedUp[s.StudentID] {
				return false
			}
		}
	}
	return true
}

func routeCost(seq []Stop) float64 {
	cost := 0.0
	for i := 1; i < len(seq); i++ {
		cost += geo.HaversineKm(seq[i-1].LatLng.Lat, seq[i-1].LatLng.Lng, seq[i].LatLng.Lat, seq[i].LatLng.Lng)
	}
	return cost
}

func BuildMapsDeepLink(seq []Stop) string {
	if len(seq) == 0 {
		return ""
	}

	destination := fmt.Sprintf("%f,%f", seq[len(seq)-1].LatLng.Lat, seq[len(seq)-1].LatLng.Lng)

	link := fmt.Sprintf("https://www.google.com/maps/dir/?api=1&destination=%s&travelmode=driving", url.QueryEscape(destination))

	if len(seq) > 1 {
		var waypoints []string
		for i := 0; i < len(seq)-1; i++ {
			waypoints = append(waypoints, fmt.Sprintf("%f,%f", seq[i].LatLng.Lat, seq[i].LatLng.Lng))
		}
		link += "&waypoints=" + url.QueryEscape(strings.Join(waypoints, "|"))
	}

	return link
}

func FormatDispatchMessage(seq []Stop, mapsLink string) string {
	var sb strings.Builder
	for _, s := range seq {
		if s.Type == Pickup {
			sb.WriteString(fmt.Sprintf("▶ Pickup: %s\n", s.Name))
		} else {
			sb.WriteString(fmt.Sprintf("⏹ Dropoff: %s\n", s.Name))
		}
	}
	sb.WriteString(fmt.Sprintf("👉 Route: %s", mapsLink))
	return sb.String()
}

// GenerateMessage computes the optimal sequence for a group of ride requests,
// builds the maps deep link, and formats the dispatch text message.
func GenerateMessage(members []models.RideRequest) ([]Stop, string, string, error) {
	var stops []Stop
	for _, m := range members {
		stops = append(stops, Stop{
			StudentID: m.ID,
			Name:      m.RequesterName,
			LatLng:    geo.LatLng{Lat: m.PickupLat, Lng: m.PickupLng},
			Type:      Pickup,
		})
		stops = append(stops, Stop{
			StudentID: m.ID,
			Name:      m.RequesterName,
			LatLng:    geo.LatLng{Lat: m.DropoffLat, Lng: m.DropoffLng},
			Type:      Dropoff,
		})
	}

	seq, err := OptimalStopSequence(stops)
	if err != nil {
		return nil, "", "", err
	}

	mapsLink := BuildMapsDeepLink(seq)
	msg := FormatDispatchMessage(seq, mapsLink)
	return seq, mapsLink, msg, nil
}
