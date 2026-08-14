package domain

import (
	"time"

	"github.com/google/uuid"
)

type FriendStatus string

const (
	FriendPending  FriendStatus = "pending"
	FriendAccepted FriendStatus = "accepted"
	FriendBlocked  FriendStatus = "blocked"
)

type Friend struct {
	UserID    uuid.UUID    `json:"user_id"`
	FriendID  uuid.UUID    `json:"friend_id"`
	Status    FriendStatus `json:"status"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}
