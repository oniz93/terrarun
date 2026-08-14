package run

import (
	"github.com/terrarun/backend/internal/config"
	"github.com/terrarun/backend/internal/domain"
	"github.com/terrarun/backend/pkg/geo"
)

type CheatResult string

const (
	CheatClean    CheatResult = "clean"
	CheatFlagged  CheatResult = "flagged"
	CheatRejected CheatResult = "rejected"
)

type CheatDetector struct {
	cfg *config.Config
}

func NewCheatDetector(cfg *config.Config) *CheatDetector {
	return &CheatDetector{cfg: cfg}
}

func (d *CheatDetector) Check(points []domain.GPSPoint, totalDistanceM, totalDurationS float64) CheatResult {
	for i := 0; i < len(points)-1; i++ {
		segDist := geo.Haversine(points[i].Lat, points[i].Lng, points[i+1].Lat, points[i+1].Lng)
		segDur := points[i+1].Timestamp.Sub(points[i].Timestamp).Seconds()
		if segDur <= 0 {
			continue
		}
		segSpeedKMH := (segDist / segDur) * 3.6

		if segSpeedKMH > d.cfg.MaxSegmentSpeedKMH {
			return CheatRejected
		}

		if segDist > d.cfg.MaxGPSJumpM {
			return CheatRejected
		}
	}

	if totalDistanceM > d.cfg.MinDistanceForAvgCheckM {
		avgSpeedKMH := (totalDistanceM / totalDurationS) * 3.6
		if avgSpeedKMH > d.cfg.MaxAvgSpeedKMH {
			return CheatRejected
		}
	}

	flagged := false

	if totalDurationS > 0 {
		avgSpeedKMH := (totalDistanceM / totalDurationS) * 3.6

		if avgSpeedKMH > d.cfg.FlagAvgSpeedKMH {
			flagged = true
		}
	}

	if totalDistanceM > d.cfg.FlagDistanceM {
		flagged = true
	}

	lowAcc := 0
	for _, p := range points {
		if p.HorizontalAccuracy != nil && *p.HorizontalAccuracy > d.cfg.FlagLowAccuracyM {
			lowAcc++
		}
	}
	if len(points) > 0 && float64(lowAcc)/float64(len(points)) > d.cfg.FlagLowAccuracyRatio {
		flagged = true
	}

	if hasHeartRate(points) {
		avgHR := avgHeartRate(points)
		avgSpeedMS := (totalDistanceM / totalDurationS)
		restHR := 60.0
		maxHR := 200.0
		expectedMin := restHR + (maxHR-restHR)*0.4
		if avgHR < expectedMin && avgSpeedMS > 1.5 {
			flagged = true
		}
	}

	if flagged {
		return CheatFlagged
	}
	return CheatClean
}

func hasHeartRate(points []domain.GPSPoint) bool {
	for _, p := range points {
		if p.HeartRate != nil {
			return true
		}
	}
	return false
}

func avgHeartRate(points []domain.GPSPoint) float64 {
	var sum int
	var count int
	for _, p := range points {
		if p.HeartRate != nil {
			sum += *p.HeartRate
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return float64(sum) / float64(count)
}

func cheatResultToRunStatus(r CheatResult) domain.RunStatus {
	switch r {
	case CheatClean:
		return domain.RunStatusCompleted
	case CheatFlagged:
		return domain.RunStatusFlagged
	case CheatRejected:
		return domain.RunStatusRejected
	default:
		return domain.RunStatusCompleted
	}
}
