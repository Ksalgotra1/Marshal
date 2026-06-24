package geo

import "math"

type LatLng struct {
	Lat float64
	Lng float64
}

func HaversineKm(lat1, lng1, lat2, lng2 float64) float64 {
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

func BearingDeg(from, to LatLng) float64 {
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

func CrossTrackDistanceKm(point, lineStart, lineEnd LatLng) float64 {
	d13 := HaversineKm(lineStart.Lat, lineStart.Lng, point.Lat, point.Lng)
	theta13 := BearingDeg(lineStart, point) * math.Pi / 180.0
	theta12 := BearingDeg(lineStart, lineEnd) * math.Pi / 180.0

	const R = 6371.0
	return math.Asin(math.Sin(d13/R)*math.Sin(theta13-theta12)) * R
}
