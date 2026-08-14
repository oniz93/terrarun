package domain

import (
	"time"

	"github.com/google/uuid"
)

type BotStatus string

const (
	BotStatusActive   BotStatus = "active"
	BotStatusInactive BotStatus = "inactive"
)

type Bot struct {
	ID             uuid.UUID  `json:"id"`
	DisplayName    string     `json:"display_name"`
	Faction        Faction    `json:"faction"`
	AvatarSeed     *string    `json:"avatar_seed,omitempty"`
	HomeLat        float64    `json:"home_lat"`
	HomeLng        float64    `json:"home_lng"`
	ActiveRadiusKM float64    `json:"active_radius_km"`
	AvgDistanceM   float64    `json:"avg_distance_m"`
	TotalRuns      int        `json:"total_runs"`
	CreatedAt      time.Time  `json:"created_at"`
	DeactivatedAt  *time.Time `json:"deactivated_at,omitempty"`
}
