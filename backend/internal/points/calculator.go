package points

import (
	"math"

	"github.com/terrarun/backend/internal/config"
)

type Calculator struct {
	cfg *config.Config
}

func NewCalculator(cfg *config.Config) *Calculator {
	return &Calculator{cfg: cfg}
}

func (c *Calculator) RunnerPoints(distanceM, elevationGainM float64, territoryPoints int, social bool, friendCount int) int {
	pts := distanceM / 100 * c.cfg.RunnerPointPer100m
	pts += elevationGainM / 10 * c.cfg.RunnerPointPer10mElev
	pts += float64(territoryPoints) * c.cfg.TerritoryPointSharePct

	if social && friendCount > 0 {
		extra := c.cfg.SocialBaseMultiplier + (float64(friendCount-1) * c.cfg.SocialExtraPerFriend)
		extra = math.Min(extra, c.cfg.SocialMultiplierCap)
		pts *= 1.0 + extra
	}

	return int(math.Round(pts))
}

func XPForLevel(level int) int64 {
	return int64(100 * math.Pow(float64(level), 1.5))
}

func LevelFromXP(totalXP int64) int {
	level := 1
	for totalXP >= XPForLevel(level) {
		level++
	}
	return level - 1
}
