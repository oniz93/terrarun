package bot

import (
	"context"
	"math"
	"math/rand"
)

type RouteGenerator struct{}

func NewRouteGenerator() *RouteGenerator {
	return &RouteGenerator{}
}

type RoutePoint struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

func (g *RouteGenerator) GenerateRoute(ctx context.Context, centerLat, centerLng float64, distanceM float64) []RoutePoint {
	steps := int(distanceM / 10)
	if steps < 10 {
		steps = 10
	}

	heading := rand.Float64() * 360
	sinH := math.Sin(heading * math.Pi / 180)
	cosH := math.Cos(heading * math.Pi / 180)

	points := make([]RoutePoint, steps)
	for i := 0; i < steps; i++ {
		frac := float64(i) / float64(steps-1)
		dist := frac * distanceM
		jitterLat := (rand.Float64() - 0.5) * 0.00005
		jitterLng := (rand.Float64() - 0.5) * 0.00005

		lat := centerLat + dist*sinH/111000.0 + jitterLat
		lng := centerLng + dist*cosH/(111000.0*math.Cos(centerLat*math.Pi/180)) + jitterLng

		points[i] = RoutePoint{Lat: lat, Lng: lng}
	}

	return points
}

func (g *RouteGenerator) GenerateClosedLoop(ctx context.Context, centerLat, centerLng float64, radiusM float64, numPoints int) [][2]float64 {
	if numPoints < 3 {
		numPoints = 20
	}

	points := make([][2]float64, numPoints)
	for i := 0; i < numPoints; i++ {
		angle := 2 * math.Pi * float64(i) / float64(numPoints)
		jitter := (rand.Float64() - 0.5) * 0.2
		r := radiusM * (1 + jitter)

		lat := centerLat + r*math.Cos(angle)/111000.0
		lng := centerLng + r*math.Sin(angle)/(111000.0*math.Cos(centerLat*math.Pi/180))

		points[i] = [2]float64{lat, lng}
	}

	return points
}
