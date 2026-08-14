package territory

import "github.com/terrarun/backend/internal/domain"

type HPState struct {
	CurrentHP int
	MaxHP     int
	OwnedBy   *domain.Faction
}

type HPTransition struct {
	FromHP          int
	ToHP            int
	Flipped         bool
	TerritoryPoints int
}

func AddHP(currentHP, maxHP int) int {
	if currentHP >= maxHP {
		return maxHP
	}
	return currentHP + 1
}

func RemoveHP(currentHP int) (newHP int, flipped bool) {
	if currentHP <= 1 {
		return 1, true
	}
	return currentHP - 1, false
}

func DailyDecay(currentHP int) int {
	if currentHP <= 1 {
		return 1
	}
	return currentHP - 1
}

func ComputeCaptureTransition(hex *domain.Hex, attackerFaction domain.Faction, cfg CaptureConfig) HPTransition {
	if hex.OwnedBy == nil {
		return HPTransition{
			FromHP:          0,
			ToHP:            1,
			Flipped:         true,
			TerritoryPoints: cfg.BasePoints,
		}
	}

	if *hex.OwnedBy == attackerFaction {
		newHP := AddHP(hex.HP, cfg.MaxHP)
		return HPTransition{
			FromHP:          hex.HP,
			ToHP:            newHP,
			Flipped:         false,
			TerritoryPoints: cfg.BasePoints,
		}
	}

	newHP, flipped := RemoveHP(hex.HP)
	pts := cfg.BasePoints
	if flipped {
		pts += int(float64(cfg.BasePoints) * cfg.StealBonusPct)
	}
	return HPTransition{
		FromHP:          hex.HP,
		ToHP:            newHP,
		Flipped:         flipped,
		TerritoryPoints: pts,
	}
}

type CaptureConfig struct {
	BasePoints    int
	MaxHP         int
	StealBonusPct float64
}
