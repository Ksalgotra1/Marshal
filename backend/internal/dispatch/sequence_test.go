package dispatch

import (
	"strings"
	"testing"

	"github.com/Ksalgotra1/Marshal/internal/geo"
	"github.com/stretchr/testify/assert"
)

func TestSameDropoff(t *testing.T) {
	stops := []Stop{
		{StudentID: "A", Name: "A", Type: Pickup, LatLng: geo.LatLng{Lat: 30.0, Lng: 76.0}},
		{StudentID: "B", Name: "B", Type: Pickup, LatLng: geo.LatLng{Lat: 30.1, Lng: 76.1}},
		{StudentID: "A", Name: "DropAB", Type: Dropoff, LatLng: geo.LatLng{Lat: 30.5, Lng: 76.5}},
		{StudentID: "B", Name: "DropAB", Type: Dropoff, LatLng: geo.LatLng{Lat: 30.5, Lng: 76.5}},
	}

	seq, err := OptimalStopSequence(stops)
	assert.NoError(t, err)
	assert.Len(t, seq, 4)

	// pick up A and B before any dropoffs
	assert.Equal(t, Pickup, seq[0].Type)
	assert.Equal(t, Pickup, seq[1].Type)
	assert.Equal(t, Dropoff, seq[2].Type)
	assert.Equal(t, Dropoff, seq[3].Type)
}

func TestDifferentDropoffs(t *testing.T) {
	// A and B pickups are very close to each other.
	// A drops at Rajpura (closer). B drops at Chandigarh (further).
	stops := []Stop{
		{StudentID: "A", Name: "A", Type: Pickup, LatLng: geo.LatLng{Lat: 30.0, Lng: 76.0}},
		{StudentID: "B", Name: "B", Type: Pickup, LatLng: geo.LatLng{Lat: 30.001, Lng: 76.001}},
		{StudentID: "A", Name: "Rajpura", Type: Dropoff, LatLng: geo.LatLng{Lat: 30.5, Lng: 76.5}},
		{StudentID: "B", Name: "Chandigarh", Type: Dropoff, LatLng: geo.LatLng{Lat: 31.0, Lng: 77.0}},
	}

	seq, err := OptimalStopSequence(stops)
	assert.NoError(t, err)
	assert.Len(t, seq, 4)

	// Pickups first
	assert.True(t, seq[0].Type == Pickup)
	assert.True(t, seq[1].Type == Pickup)
	// Rajpura dropoff is closer to pickups, so it should be the 3rd stop.
	assert.Equal(t, "Rajpura", seq[2].Name)
	assert.Equal(t, "Chandigarh", seq[3].Name)
}

func TestViolatesConstraints(t *testing.T) {
	// Setup where going Dropoff -> Pickup would be spatially shorter.
	// But it violates constraints.
	stops := []Stop{
		{StudentID: "A", Name: "A", Type: Pickup, LatLng: geo.LatLng{Lat: 30.0, Lng: 76.0}},
		{StudentID: "B", Name: "B", Type: Pickup, LatLng: geo.LatLng{Lat: 30.9, Lng: 76.9}},
		{StudentID: "A", Name: "DropA", Type: Dropoff, LatLng: geo.LatLng{Lat: 30.8, Lng: 76.8}}, // DropA is right next to PickupB
		{StudentID: "B", Name: "DropB", Type: Dropoff, LatLng: geo.LatLng{Lat: 31.0, Lng: 77.0}},
	}

	seq, err := OptimalStopSequence(stops)
	assert.NoError(t, err)

	pickedUp := make(map[string]bool)
	for _, s := range seq {
		if s.Type == Pickup {
			pickedUp[s.StudentID] = true
		} else {
			assert.True(t, pickedUp[s.StudentID], "Dropoff before pickup!")
		}
	}
}

func TestMapsLinkStructure(t *testing.T) {
	seq := []Stop{
		{LatLng: geo.LatLng{Lat: 30.1, Lng: 76.1}},
		{LatLng: geo.LatLng{Lat: 30.2, Lng: 76.2}},
		{LatLng: geo.LatLng{Lat: 30.3, Lng: 76.3}},
	}
	link := BuildMapsDeepLink(30.0, 76.0, seq)

	assert.Contains(t, link, "origin=30.000000%2C76.000000")
	assert.Contains(t, link, "destination=30.300000%2C76.300000")
	assert.Contains(t, link, "waypoints=30.100000%2C76.100000%7C30.200000%2C76.200000")
	assert.Contains(t, link, "travelmode=driving")
}

func TestDispatchMessageFormat(t *testing.T) {
	seq := []Stop{
		{Type: Pickup, Name: "Alice"},
		{Type: Dropoff, Name: "Alice Drop"},
	}
	msg := FormatDispatchMessage(seq, "http://maps.link")

	assert.True(t, strings.HasPrefix(msg, "▶ Pickup: Alice\n"))
	assert.Contains(t, msg, "⏹ Dropoff: Alice Drop\n")
	assert.True(t, strings.HasSuffix(msg, "👉 Route: http://maps.link"))
}
