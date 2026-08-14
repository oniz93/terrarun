package h3util

import (
	"math"

	"github.com/uber/h3-go/v4"
)

const DefaultResolution = 11

func LatLngToCell(lat, lng float64, res int) (h3.Cell, error) {
	return h3.LatLngToCell(h3.LatLng{Lat: lat, Lng: lng}, res)
}

func CellToLatLng(cell h3.Cell) (float64, float64, error) {
	ll, err := h3.CellToLatLng(cell)
	if err != nil {
		return 0, 0, err
	}
	return ll.Lat, ll.Lng, nil
}

func PolygonToCells(vertices [][2]float64, res int) ([]h3.Cell, error) {
	if len(vertices) < 3 {
		return nil, nil
	}

	loop := make([]h3.LatLng, len(vertices))
	for i, v := range vertices {
		loop[i] = h3.LatLng{Lat: v[0], Lng: v[1]}
	}

	return h3.PolygonToCells(h3.GeoPolygon{
		GeoLoop: h3.GeoLoop(loop),
	}, res)
}

func GridDisk(cell h3.Cell, k int) ([]h3.Cell, error) {
	return h3.GridDisk(cell, k)
}

func IsCellInPolygon(cell h3.Cell, polygon [][2]float64) bool {
	lat, lng, err := CellToLatLng(cell)
	if err != nil {
		return false
	}
	return pointInPolygon(lat, lng, polygon)
}

func pointInPolygon(lat, lng float64, polygon [][2]float64) bool {
	inside := false
	n := len(polygon)
	j := n - 1
	for i := 0; i < n; i++ {
		if ((polygon[i][1] > lng) != (polygon[j][1] > lng)) &&
			(lat < (polygon[j][0]-polygon[i][0])*(lng-polygon[i][1])/(polygon[j][1]-polygon[i][1])+polygon[i][0]) {
			inside = !inside
		}
		j = i
	}
	return inside
}

func H3IndexToInt64(cell h3.Cell) int64 {
	return int64(cell)
}

func Int64ToH3Index(val int64) h3.Cell {
	return h3.Cell(val)
}

func DiskDistance(res int, radiusKM float64) int {
	area, err := h3.HexagonAreaAvgKm2(res)
	if err != nil || area == 0 {
		return 1
	}
	k := int(math.Ceil(math.Sqrt(radiusKM*radiusKM / area)))
	if k < 1 {
		return 1
	}
	return k
}

func BBoxToCells(minLat, minLng, maxLat, maxLng float64, res int) ([]h3.Cell, error) {
	centerCell, err := LatLngToCell((minLat+maxLat)/2, (minLng+maxLng)/2, res)
	if err != nil {
		return nil, err
	}

	center, err := h3.CellToLatLng(centerCell)
	if err != nil {
		return nil, err
	}
	minDist := hav(center.Lat, center.Lng, minLat, minLng)
	maxDist := hav(center.Lat, center.Lng, maxLat, maxLng)
	radiusKM := math.Max(minDist, maxDist) / 1000.0

	k := DiskDistance(res, radiusKM)
	return h3.GridDisk(centerCell, k)
}

func hav(lat1, lng1, lat2, lng2 float64) float64 {
	dLat := (lat2 - lat1) * math.Pi / 180
	dLng := (lng2 - lng1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return 6371000 * c
}
