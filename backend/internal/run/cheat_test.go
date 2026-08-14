package run

import (
	"testing"
	"time"

	"github.com/terrarun/backend/internal/config"
	"github.com/terrarun/backend/internal/domain"
)

func newTestDetector(t *testing.T) *CheatDetector {
	t.Helper()
	cfg := &config.Config{
		MaxSegmentSpeedKMH:      35,
		MaxAvgSpeedKMH:          20,
		MinDistanceForAvgCheckM: 2000,
		MaxGPSJumpM:             500,
		FlagAvgSpeedKMH:         15,
		FlagDistanceM:           50000,
		FlagLowAccuracyM:        20,
		FlagLowAccuracyRatio:    0.3,
	}
	return NewCheatDetector(cfg)
}

func mkPoint(lat, lng float64, t time.Time, acc *float64, hr *int) domain.GPSPoint {
	return domain.GPSPoint{
		Lat:                lat,
		Lng:                lng,
		Timestamp:          t,
		HorizontalAccuracy: acc,
		HeartRate:          hr,
	}
}

func TestCheatClean(t *testing.T) {
	d := newTestDetector(t)
	now := time.Now()
	points := []domain.GPSPoint{
		mkPoint(51.5074, -0.1278, now, ptr(5.0), nil),
		mkPoint(51.5075, -0.1276, now.Add(10*time.Second), ptr(5.0), nil),
		mkPoint(51.5076, -0.1274, now.Add(20*time.Second), ptr(5.0), nil),
	}
	result := d.Check(points, 30.0, 20.0)
	if result != CheatClean {
		t.Errorf("expected clean, got %s", result)
	}
}

func TestCheatRejected_SegmentSpeed(t *testing.T) {
	d := newTestDetector(t)
	now := time.Now()
	points := []domain.GPSPoint{
		mkPoint(51.5074, -0.1278, now, nil, nil),
		mkPoint(51.5174, -0.1178, now.Add(1*time.Second), nil, nil),
	}
	result := d.Check(points, 0, 0)
	if result != CheatRejected {
		t.Errorf("expected rejected (segment speed >35km/h), got %s", result)
	}
}

func TestCheatRejected_AvgSpeed(t *testing.T) {
	d := newTestDetector(t)
	now := time.Now()
	points := []domain.GPSPoint{
		mkPoint(51.5074, -0.1278, now, nil, nil),
		mkPoint(51.5075, -0.1276, now.Add(10*time.Second), nil, nil),
		mkPoint(51.5076, -0.1274, now.Add(20*time.Second), nil, nil),
	}
	result := d.Check(points, 5000, 500)
	if result != CheatRejected {
		t.Errorf("expected rejected (avg speed >20km/h), got %s", result)
	}
}

func TestCheatRejected_GPSJump(t *testing.T) {
	d := newTestDetector(t)
	now := time.Now()
	points := []domain.GPSPoint{
		mkPoint(51.5074, -0.1278, now, nil, nil),
		mkPoint(52.0, -1.0, now.Add(10*time.Second), nil, nil),
	}
	result := d.Check(points, 0, 0)
	if result != CheatRejected {
		t.Errorf("expected rejected (GPS jump >500m), got %s", result)
	}
}

func TestCheatFlagged_AvgSpeed(t *testing.T) {
	d := newTestDetector(t)
	now := time.Now()
	points := []domain.GPSPoint{
		mkPoint(51.5074, -0.1278, now, ptr(5.0), nil),
		mkPoint(51.5075, -0.1276, now.Add(30*time.Second), ptr(5.0), nil),
	}
	result := d.Check(points, 200, 60)
	if result != CheatClean {
		t.Errorf("expected clean (200m in 60s = 12km/h, below flag), got %s", result)
	}
}

func TestCheatFlagged_DistanceTooLong(t *testing.T) {
	d := newTestDetector(t)
	now := time.Now()
	points := []domain.GPSPoint{
		mkPoint(51.5074, -0.1278, now, ptr(5.0), nil),
		mkPoint(51.5075, -0.1276, now.Add(30*time.Second), ptr(5.0), nil),
	}
	result := d.Check(points, 60000, 36000)
	if result != CheatFlagged {
		t.Errorf("expected flagged (distance >50km), got %s", result)
	}
}

func TestCheatFlagged_LowAccuracy(t *testing.T) {
	d := newTestDetector(t)
	now := time.Now()
	points := []domain.GPSPoint{
		mkPoint(51.5074, -0.1278, now, ptr(50.0), nil),
		mkPoint(51.5075, -0.1276, now.Add(10*time.Second), ptr(60.0), nil),
		mkPoint(51.5076, -0.1274, now.Add(20*time.Second), ptr(5.0), nil),
	}
	result := d.Check(points, 100, 20)
	if result != CheatFlagged {
		t.Errorf("expected flagged (>30%% low accuracy), got %s", result)
	}
}

func TestCheatFlagged_HRMismatch(t *testing.T) {
	d := newTestDetector(t)
	now := time.Now()
	hr := 50
	points := []domain.GPSPoint{
		mkPoint(51.5074, -0.1278, now, ptr(5.0), &hr),
		mkPoint(51.5075, -0.1276, now.Add(10*time.Second), ptr(5.0), &hr),
		mkPoint(51.5076, -0.1274, now.Add(20*time.Second), ptr(5.0), &hr),
	}
	result := d.Check(points, 200, 20)
	if result != CheatFlagged {
		t.Errorf("expected flagged (low HR for speed), got %s", result)
	}
}

func ptr(f float64) *float64 {
	return &f
}
