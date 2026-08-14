package run

import (
	"math"

	"github.com/terrarun/backend/internal/domain"
	"github.com/terrarun/backend/pkg/geo"
)

type RunStats struct {
	DistanceM     float64
	DurationS     float64
	ElevationGain float64
	AvgPace       float64
	MaxSpeed      float64
	AvgHR         float64
	MaxHR         float64
	Calories      int
	Polyline      string
}

type StatsCalculator struct{}

func NewStatsCalculator() *StatsCalculator {
	return &StatsCalculator{}
}

func (c *StatsCalculator) Compute(points []domain.GPSPoint) RunStats {
	if len(points) < 2 {
		return RunStats{}
	}

	start := points[0].Timestamp
	end := points[len(points)-1].Timestamp
	durationS := end.Sub(start).Seconds()

	totalDist := geo.TotalDistance(points)
	elevGain := geo.ElevationGain(points)
	avgPace := geo.AvgPace(totalDist, durationS)
	maxSpeed := geo.MaxSpeed(points)

	avgHR, maxHR := computeHRStats(points)
	calories := estimateCalories(totalDist, elevGain, avgHR)

	simplified := geo.DouglasPeucker(points, 10.0)
	polyline := geo.NewPolylineEncoder(simplified).Encode()

	return RunStats{
		DistanceM:     math.Round(totalDist*100) / 100,
		DurationS:     math.Round(durationS*100) / 100,
		ElevationGain: math.Round(elevGain*100) / 100,
		AvgPace:       math.Round(avgPace*100) / 100,
		MaxSpeed:      math.Round(maxSpeed*100) / 100,
		AvgHR:         avgHR,
		MaxHR:         maxHR,
		Calories:      calories,
		Polyline:      polyline,
	}
}

func computeHRStats(points []domain.GPSPoint) (avg, max float64) {
	var sum int
	var count int
	for _, p := range points {
		if p.HeartRate != nil {
			hr := *p.HeartRate
			sum += hr
			count++
			if float64(hr) > max {
				max = float64(hr)
			}
		}
	}
	if count == 0 {
		return 0, 0
	}
	return float64(sum) / float64(count), max
}

func estimateCalories(distanceM, elevationGain, avgHR float64) int {
	distanceKM := distanceM / 1000
	met := 8.0
	if avgHR > 0 {
		met = 1.0 + (avgHR-60)/20.0
		if met < 3 {
			met = 3
		}
		if met > 16 {
			met = 16
		}
	}
	weightKG := 70.0
	hours := distanceKM / 10.0
	if hours <= 0 {
		hours = 0.1
	}
	cal := met * weightKG * hours
	return int(math.Round(cal))
}
