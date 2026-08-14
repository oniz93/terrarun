package geo

import (
	"math"

	"github.com/terrarun/backend/internal/domain"
)

const earthRadiusM = 6371000

func Haversine(lat1, lng1, lat2, lng2 float64) float64 {
	dLat := (lat2 - lat1) * math.Pi / 180
	dLng := (lng2 - lng1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusM * c
}

func HaversineKM(lat1, lng1, lat2, lng2 float64) float64 {
	return Haversine(lat1, lng1, lat2, lng2) / 1000.0
}

type BBox struct {
	MinLat float64
	MinLng float64
	MaxLat float64
	MaxLng float64
}

func BBoxForRadius(centerLat, centerLng, radiusKM float64) BBox {
	latChange := (radiusKM / 111.32)
	lngChange := (radiusKM / (111.32 * math.Cos(centerLat*math.Pi/180)))
	return BBox{
		MinLat: centerLat - latChange,
		MaxLat: centerLat + latChange,
		MinLng: centerLng - lngChange,
		MaxLng: centerLng + lngChange,
	}
}

func DouglasPeucker(points []domain.GPSPoint, epsilonM float64) []domain.GPSPoint {
	if len(points) <= 2 {
		return points
	}

	dmax := 0.0
	index := 0
	end := len(points) - 1

	for i := 1; i < end; i++ {
		d := perpendicularDistance(
			points[i].Lat, points[i].Lng,
			points[0].Lat, points[0].Lng,
			points[end].Lat, points[end].Lng,
		)
		if d > dmax {
			index = i
			dmax = d
		}
	}

	if dmax > epsilonM {
		left := DouglasPeucker(points[:index+1], epsilonM)
		right := DouglasPeucker(points[index:], epsilonM)
		return append(left[:len(left)-1], right...)
	}

	return []domain.GPSPoint{points[0], points[end]}
}

func perpendicularDistance(plat, plng, l1lat, l1lng, l2lat, l2lng float64) float64 {
	x0 := plng
	y0 := plat
	x1 := l1lng
	y1 := l1lat
	x2 := l2lng
	y2 := l2lat

	num := math.Abs((y2-y1)*x0 - (x2-x1)*y0 + x2*y1 - y2*x1)
	den := math.Sqrt((y2-y1)*(y2-y1) + (x2-x1)*(x2-x1))
	if den == 0 {
		return Haversine(plat, plng, l1lat, l1lng)
	}
	return num / den * 111320
}

func TotalDistance(points []domain.GPSPoint) float64 {
	var total float64
	for i := 0; i < len(points)-1; i++ {
		total += Haversine(points[i].Lat, points[i].Lng, points[i+1].Lat, points[i+1].Lng)
	}
	return total
}

func ElevationGain(points []domain.GPSPoint) float64 {
	if len(points) == 0 {
		return 0
	}
	var gain float64
	for i := 1; i < len(points); i++ {
		if points[i].Altitude != nil && points[i-1].Altitude != nil {
			diff := *points[i].Altitude - *points[i-1].Altitude
			if diff > 0 {
				gain += diff
			}
		}
	}
	return gain
}

func AvgPace(distanceM, durationS float64) float64 {
	if distanceM <= 0 {
		return 0
	}
	return durationS / (distanceM / 1000.0)
}

func MaxSpeed(points []domain.GPSPoint) float64 {
	var maxSpeed float64
	for i := 0; i < len(points)-1; i++ {
		d := Haversine(points[i].Lat, points[i].Lng, points[i+1].Lat, points[i+1].Lng)
		t := points[i+1].Timestamp.Sub(points[i].Timestamp).Seconds()
		if t <= 0 {
			continue
		}
		speedMS := d / t
		speedKMH := speedMS * 3.6
		if speedKMH > maxSpeed {
			maxSpeed = speedKMH
		}
	}
	return maxSpeed
}

type PolylineEncoder struct {
	points []domain.GPSPoint
}

func NewPolylineEncoder(points []domain.GPSPoint) *PolylineEncoder {
	return &PolylineEncoder{points: points}
}

func (e *PolylineEncoder) Encode() string {
	if len(e.points) == 0 {
		return ""
	}
	var result []byte
	prevLat := 0
	prevLng := 0

	for _, p := range e.points {
		lat := int(math.Round(p.Lat * 1e5))
		lng := int(math.Round(p.Lng * 1e5))
		result = append(result, encodeSigned(lat-prevLat)...)
		result = append(result, encodeSigned(lng-prevLng)...)
		prevLat = lat
		prevLng = lng
	}
	return string(result)
}

func encodeSigned(value int) []byte {
	value = value << 1
	if value < 0 {
		value = ^value
	}
	var result []byte
	for value >= 0x20 {
		result = append(result, byte(0x20|(value&0x1f))+63)
		value >>= 5
	}
	result = append(result, byte(value)+63)
	return result
}

func PolygonArea(vertices [][2]float64) float64 {
	n := len(vertices)
	if n < 3 {
		return 0
	}
	var area float64
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		area += vertices[i][0] * vertices[j][1]
		area -= vertices[j][0] * vertices[i][1]
	}
	return math.Abs(area) / 2.0
}

func ExtractLargestSimplePolygon(vertices [][2]float64) [][2]float64 {
	if len(vertices) < 3 || !isSelfIntersecting(vertices) {
		return vertices
	}
	return simplifySelfIntersecting(vertices)
}

func isSelfIntersecting(vertices [][2]float64) bool {
	n := len(vertices)
	for i := 0; i < n; i++ {
		p1 := vertices[i]
		p2 := vertices[(i+1)%n]
		for j := i + 2; j < n; j++ {
			if (j+1)%n == i {
				continue
			}
			p3 := vertices[j]
			p4 := vertices[(j+1)%n]
			if segmentsIntersect(p1, p2, p3, p4) {
				return true
			}
		}
	}
	return false
}

func segmentsIntersect(p1, p2, p3, p4 [2]float64) bool {
	d1 := cross(p3, p4, p1)
	d2 := cross(p3, p4, p2)
	d3 := cross(p1, p2, p3)
	d4 := cross(p1, p2, p4)
	if ((d1 > 0 && d2 < 0) || (d1 < 0 && d2 > 0)) &&
		((d3 > 0 && d4 < 0) || (d3 < 0 && d4 > 0)) {
		return true
	}
	return false
}

func cross(a, b, c [2]float64) float64 {
	return (b[0]-a[0])*(c[1]-a[1]) - (b[1]-a[1])*(c[0]-a[0])
}

func simplifySelfIntersecting(vertices [][2]float64) [][2]float64 {
	n := len(vertices)
	bestArea := 0.0
	bestPoly := vertices

	for i := 0; i < n-2; i++ {
		for j := i + 2; j < n; j++ {
			candidate := append([][2]float64{}, vertices[i:j+1]...)
			if !isSelfIntersecting(candidate) && len(candidate) >= 3 {
				area := PolygonArea(candidate)
				if area > bestArea {
					bestArea = area
					bestPoly = candidate
				}
			}
		}
	}
	return bestPoly
}

func PointsToPolygonRing(points []domain.GPSPoint) [][2]float64 {
	if len(points) < 3 {
		return nil
	}
	ring := make([][2]float64, len(points)+1)
	for i, p := range points {
		ring[i] = [2]float64{p.Lat, p.Lng}
	}
	ring[len(points)] = [2]float64{points[0].Lat, points[0].Lng}
	return ring
}

func BuildPathCorridor(points []domain.GPSPoint, bufferM float64) [][2]float64 {
	if len(points) < 2 {
		return nil
	}

	bufferDeg := bufferM / 111320.0
	var corridor [][2]float64

	for i := len(points) - 1; i >= 0; i-- {
		corridor = append(corridor, [2]float64{
			points[i].Lat + bufferDeg,
			points[i].Lng + bufferDeg,
		})
	}
	for _, p := range points {
		corridor = append(corridor, [2]float64{
			p.Lat - bufferDeg,
			p.Lng - bufferDeg,
		})
	}
	return corridor
}

func IsPointInPolygon(lat, lng float64, polygon [][2]float64) bool {
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

func AvgSpeed(points []domain.GPSPoint) float64 {
	if len(points) < 2 {
		return 0
	}
	totalDist := TotalDistance(points)
	totalDur := points[len(points)-1].Timestamp.Sub(points[0].Timestamp).Seconds()
	if totalDur <= 0 {
		return 0
	}
	return (totalDist / totalDur) * 3.6
}
