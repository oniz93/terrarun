package domain

import (
	"time"

	"github.com/google/uuid"
)

type Season struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	StartsAt  time.Time `json:"starts_at"`
	EndsAt    time.Time `json:"ends_at"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SeasonParticipant struct {
	UserID      uuid.UUID `json:"user_id"`
	SeasonID    uuid.UUID `json:"season_id"`
	Faction     Faction   `json:"faction"`
	Points      int       `json:"points"`
	DisplayName string    `json:"display_name"`
}

type TiersConfig struct {
	Rookie   int `json:"rookie"`
	Runner   int `json:"runner"`
	Sprinter int `json:"sprinter"`
	Elite    int `json:"elite"`
	Legend   int `json:"legend"`
}
