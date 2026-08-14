package notification

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	"github.com/terrarun/backend/internal/websocket"
)

type Service struct {
	db  *pgxpool.Pool
	rdb *redis.Client
	hub *websocket.Hub
}

func NewService(db *pgxpool.Pool, rdb *redis.Client, hub *websocket.Hub) *Service {
	return &Service{db: db, rdb: rdb, hub: hub}
}

type NotificationType string

const (
	NotifTerritoryLost   NotificationType = "territory_lost"
	NotifTerritoryGained NotificationType = "territory_gained"
	NotifFriendRequest   NotificationType = "friend_request"
	NotifFriendAccepted  NotificationType = "friend_accepted"
	NotifRunMilestone    NotificationType = "run_milestone"
	NotifSeasonStart     NotificationType = "season_start"
	NotifSeasonEnd       NotificationType = "season_end"
	NotifLevelUp         NotificationType = "level_up"
)

type Notification struct {
	ID        uuid.UUID        `json:"id"`
	UserID    uuid.UUID        `json:"user_id"`
	Type      NotificationType `json:"type"`
	Title     string           `json:"title"`
	Body      string           `json:"body"`
	Data      interface{}      `json:"data,omitempty"`
	CreatedAt string           `json:"created_at"`
	Read      bool             `json:"read"`
}

func (s *Service) SendNotification(ctx context.Context, userID uuid.UUID, nType NotificationType, title, body string, data interface{}) error {
	notif := &Notification{
		ID:        uuid.New(),
		UserID:    userID,
		Type:      nType,
		Title:     title,
		Body:      body,
		Data:      data,
		CreatedAt: "now()",
		Read:      false,
	}

	dataBytes, err := json.Marshal(notif.Data)
	if err != nil {
		return fmt.Errorf("marshal notif data: %w", err)
	}

	_, err = s.db.Exec(ctx,
		`INSERT INTO notifications (id, user_id, type, title, body, data)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		notif.ID, notif.UserID, string(notif.Type), notif.Title, notif.Body, dataBytes)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID.String()).Msg("failed to store notification")
		return fmt.Errorf("store notification: %w", err)
	}

	s.hub.SendToUser(userID, &websocket.Message{
		Type:    "notification",
		Payload: notif,
	})

	log.Info().Str("user_id", userID.String()).Str("type", string(nType)).Msg("notification sent")
	return nil
}

func (s *Service) GetUnread(ctx context.Context, userID uuid.UUID) ([]Notification, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, user_id, type, title, body, data, created_at, read
		 FROM notifications WHERE user_id = $1 AND read = false
		 ORDER BY created_at DESC LIMIT 50`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifs []Notification
	for rows.Next() {
		var n Notification
		var dataBytes []byte
		if err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Body, &dataBytes, &n.CreatedAt, &n.Read); err != nil {
			return nil, err
		}
		if dataBytes != nil {
			json.Unmarshal(dataBytes, &n.Data)
		}
		notifs = append(notifs, n)
	}
	return notifs, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, notifID uuid.UUID) error {
	_, err := s.db.Exec(ctx,
		`UPDATE notifications SET read = true WHERE id = $1 AND user_id = $2`,
		notifID, userID)
	return err
}

func (s *Service) Delete(ctx context.Context, userID, notifID uuid.UUID) error {
	_, err := s.db.Exec(ctx,
		`DELETE FROM notifications WHERE id = $1 AND user_id = $2`,
		notifID, userID)
	return err
}

func (s *Service) NotifyTerritoryGained(ctx context.Context, userID uuid.UUID, hexCount int) {
	s.SendNotification(ctx, userID, NotifTerritoryGained,
		"Territory Captured!",
		fmt.Sprintf("Your team captured %d new hexes", hexCount),
		map[string]interface{}{"hex_count": hexCount})
}

func (s *Service) NotifyTerritoryLost(ctx context.Context, userID uuid.UUID, hexCount int) {
	s.SendNotification(ctx, userID, NotifTerritoryLost,
		"Territory Lost",
		fmt.Sprintf("Your team lost %d hexes", hexCount),
		map[string]interface{}{"hex_count": hexCount})
}

func (s *Service) NotifyLevelUp(ctx context.Context, userID uuid.UUID, newLevel int) {
	s.SendNotification(ctx, userID, NotifLevelUp,
		"Level Up!",
		fmt.Sprintf("Congratulations! You reached level %d", newLevel),
		map[string]interface{}{"level": newLevel})
}
