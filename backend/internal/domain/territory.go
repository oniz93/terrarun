package domain

import (
	"time"

	"github.com/google/uuid"
)

type Hex struct {
	H3Index       int64      `json:"h3_index"`
	OwnedBy       *Faction   `json:"owned_by,omitempty"`
	HP            int        `json:"hp"`
	CapturedByID  *uuid.UUID `json:"-"`
	CapturedAt    time.Time  `json:"captured_at"`
	LastDecayedAt time.Time  `json:"-"`
	IsNatural     bool       `json:"is_natural"`
	Lat           float64    `json:"-"`
	Lng           float64    `json:"-"`
}

type TerritoryChange struct {
	ID            int64      `json:"id"`
	H3Index       int64      `json:"h3_index"`
	RunID         *uuid.UUID `json:"run_id,omitempty"`
	PreviousOwner *Faction   `json:"previous_owner,omitempty"`
	NewOwner      *Faction   `json:"new_owner,omitempty"`
	HPBefore      int        `json:"hp_before"`
	HPAfter       int        `json:"hp_after"`
	ChangedAt     time.Time  `json:"changed_at"`
}

type TerritoryStats struct {
	TotalHexes      int64   `json:"total_hexes"`
	ClaimedHexes    int64   `json:"claimed_hexes"`
	NeonHexes       int64   `json:"neon_hexes"`
	UmbraHexes      int64   `json:"umbra_hexes"`
	NeonPercentage  float64 `json:"neon_percentage"`
	UmbraPercentage float64 `json:"umbra_percentage"`
	ContestedZones  int64   `json:"contested_zones"`
}
