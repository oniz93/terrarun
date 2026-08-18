package points

import (
	"math"
	"testing"

	"github.com/terrarun/backend/internal/config"
)

func newTestCalc(t *testing.T) *Calculator {
	t.Helper()
	cfg := &config.Config{
		RunnerPointPer100m:     1.0,
		RunnerPointPer10mElev:  5.0,
		TerritoryPointSharePct: 0.1,
		SocialBaseMultiplier:   0.25,
		SocialExtraPerFriend:   0.10,
		SocialMultiplierCap:    0.50,
	}
	return NewCalculator(cfg)
}

func TestRunnerPoints_DistanceOnly(t *testing.T) {
	c := newTestCalc(t)
	pts := c.RunnerPoints(5000, 0, 0, false, 0)
	if pts != 50 {
		t.Errorf("expected 50 pts for 5000m, got %d", pts)
	}
}

func TestRunnerPoints_DistanceAndElevation(t *testing.T) {
	c := newTestCalc(t)
	pts := c.RunnerPoints(5000, 100, 0, false, 0)
	expected := 50 + (100/10)*5
	if pts != expected {
		t.Errorf("expected %d pts, got %d", expected, pts)
	}
}

func TestRunnerPoints_TerritoryShare(t *testing.T) {
	c := newTestCalc(t)
	pts := c.RunnerPoints(5000, 0, 100, false, 0)
	expected := 50 + int(100*0.1)
	if pts != expected {
		t.Errorf("expected %d pts, got %d", expected, pts)
	}
}

func TestRunnerPoints_SocialMultiplier(t *testing.T) {
	c := newTestCalc(t)
	pts := c.RunnerPoints(5000, 0, 0, true, 2)
	base := 50
	multiplier := 1.0 + 0.25 + (2-1)*0.10
	expected := int(math.Round(float64(base) * multiplier))
	if pts != expected {
		t.Errorf("social run with 2 friends: expected %d, got %d (base=%d, mult=%.2f)", expected, pts, base, multiplier)
	}
}

func TestRunnerPoints_SocialMultiplierCapped(t *testing.T) {
	c := newTestCalc(t)
	pts := c.RunnerPoints(5000, 0, 0, true, 20)
	base := 50
	multiplier := 1.0 + 0.50
	expected := int(float64(base) * multiplier)
	if pts != expected {
		t.Errorf("capped social: expected %d, got %d", expected, pts)
	}
}

func TestRunnerPoints_SoloNoMultiplier(t *testing.T) {
	c := newTestCalc(t)
	pts := c.RunnerPoints(5000, 0, 0, false, 5)
	if pts != 50 {
		t.Errorf("solo run should have no multiplier, expected 50, got %d", pts)
	}
}

func TestXPForLevel(t *testing.T) {
	tests := []struct {
		level int
		xp    int64
	}{
		{1, 0},
		{2, 100},
		{3, 282},
		{4, 519},
		{5, 800},
		{6, 1118},
		{11, 3162},
	}
	for _, tt := range tests {
		got := XPForLevel(tt.level)
		if got != tt.xp {
			t.Errorf("XPForLevel(%d) = %d, want %d", tt.level, got, tt.xp)
		}
	}
}

func TestLevelFromXP(t *testing.T) {
	tests := []struct {
		xp    int64
		level int
	}{
		{0, 1},
		{50, 1},
		{100, 2},
		{281, 2},
		{282, 3},
		{5000, 14},
		{100000, 101},
	}
	for _, tt := range tests {
		got := LevelFromXP(tt.xp)
		if got != tt.level {
			t.Errorf("LevelFromXP(%d) = %d, want %d", tt.xp, got, tt.level)
		}
	}
}

func TestRunnerPoints_AllComponents(t *testing.T) {
	c := newTestCalc(t)
	distanceM := 10000.0
	elevationGainM := 50.0
	territoryPoints := 200
	social := true
	friendCount := 3

	baseDist := distanceM / 100 * c.cfg.RunnerPointPer100m
	baseElev := elevationGainM / 10 * c.cfg.RunnerPointPer10mElev
	baseTerr := float64(territoryPoints) * c.cfg.TerritoryPointSharePct
	base := baseDist + baseElev + baseTerr
	mult := 1.0 + c.cfg.SocialBaseMultiplier + float64(friendCount-1)*c.cfg.SocialExtraPerFriend
	expected := int(base * mult)

	pts := c.RunnerPoints(distanceM, elevationGainM, territoryPoints, social, friendCount)
	if pts != expected {
		t.Errorf("expected %d, got %d", expected, pts)
	}
}
