package domain

import (
	"time"

	"github.com/google/uuid"
)

type Season struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

type SeasonParticipant struct {
	UserID          uuid.UUID `json:"user_id"`
	SeasonID        uuid.UUID `json:"season_id"`
	Faction         Faction   `json:"faction"`
	RunnerPoints    int       `json:"runner_points"`
	TerritoryPoints int       `json:"territory_points"`
	CurrentTier     string    `json:"current_tier"`
	DisplayName     string    `json:"display_name"`
}

type TiersConfig struct {
	Rookie   int `json:"rookie"`
	Runner   int `json:"runner"`
	Sprinter int `json:"sprinter"`
	Elite    int `json:"elite"`
	Legend   int `json:"legend"`
}
