package main

import (
	"fmt"
	"math"

	"github.com/uber/h3-go/v4"
)

// Haversine formula from geo package to check distances
func haversineKm(lat1, lon1, lat2, lon2 float64) float64 {
	const r = 6371.0
	lat1 = lat1 * math.Pi / 180
	lat2 = lat2 * math.Pi / 180
	lon1 = lon1 * math.Pi / 180
	lon2 = lon2 * math.Pi / 180

	dlat := lat2 - lat1
	dlon := lon2 - lon1

	a := math.Sin(dlat/2)*math.Sin(dlat/2) + math.Cos(lat1)*math.Cos(lat2)*math.Sin(dlon/2)*math.Sin(dlon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return r * c
}

func main() {
	// Let's generate a point at a specific distance
	lat1, lng1 := 30.7414, 76.7680 // Sector 17
	
	// approximate 1 km in degrees lat is ~0.009
	// lat2 is lat1 + (3.5 / 111.32)
	lat2_35km, lng2_35km := lat1 + (3.5 / 111.32), lng1
	lat2_28km, lng2_28km := lat1 + (2.8 / 111.32), lng1
	
	pairs := []struct{l1, lg1, l2, lg2 float64}{
		{lat1, lng1, lat2_35km, lng2_35km},
		{lat1, lng1, lat2_28km, lng2_28km},
	}
	
	for i, p := range pairs {
		dist := haversineKm(p.l1, p.lg1, p.l2, p.lg2)
		fmt.Printf("Pair %d - Physical Distance: %.2f km\n", i, dist)
		for res := 7; res <= 9; res++ {
			cell1, _ := h3.LatLngToCell(h3.LatLng{Lat: p.l1, Lng: p.lg1}, res)
			cell2, _ := h3.LatLngToCell(h3.LatLng{Lat: p.l2, Lng: p.lg2}, res)
			
			ring2, _ := h3.GridDisk(cell1, 2)
			found := false
			for _, c := range ring2 {
				if c == cell2 { found = true; break }
			}
			fmt.Printf("  Res %d in GridDisk(2)? %v\n", res, found)
		}
	}
}
