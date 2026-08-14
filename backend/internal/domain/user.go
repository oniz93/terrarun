package domain

import (
	"time"

	"github.com/google/uuid"
)

type Faction string

const (
	FactionNeon  Faction = "neon"
	FactionUmbra Faction = "umbra"
)

type RunnerTier string

const (
	TierFree    RunnerTier = "free"
	TierRunner  RunnerTier = "runner"
	TierCaptain RunnerTier = "captain"
)

type User struct {
	ID           uuid.UUID  `json:"id"`
	Email        string     `json:"email"`
	PasswordHash *string    `json:"-"`
	DisplayName  string     `json:"display_name"`
	PhoneHash    *string    `json:"phone_hash,omitempty"`
	AvatarURL    *string    `json:"avatar_url,omitempty"`
	Faction      *Faction   `json:"faction,omitempty"`
	AccountLevel int        `json:"account_level"`
	AccountXP    int64      `json:"account_xp"`
	GoogleID     *string    `json:"-"`
	AppleID      *string    `json:"-"`
	RunnerTier   RunnerTier `json:"runner_tier"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"-"`
}

type Session struct {
	ID           uuid.UUID  `json:"id"`
	UserID       uuid.UUID  `json:"user_id"`
	RefreshToken string     `json:"refresh_token"`
	DeviceInfo   *string    `json:"device_info,omitempty"`
	ExpiresAt    time.Time  `json:"expires_at"`
	CreatedAt    time.Time  `json:"created_at"`
	RevokedAt    *time.Time `json:"-"`
}
