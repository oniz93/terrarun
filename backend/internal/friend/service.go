package friend

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	"github.com/terrarun/backend/internal/domain"
	"github.com/terrarun/backend/internal/user"
)

func avatarURL(u *domain.User) string {
	if u.AvatarURL != nil {
		return *u.AvatarURL
	}
	return ""
}

var (
	ErrAlreadyFriends = errors.New("already friends")
	ErrSelfRequest    = errors.New("cannot friend yourself")
	ErrNotFound       = errors.New("friend request not found")
	ErrBlocked        = errors.New("request blocked")
)

type Service struct {
	repo     Repository
	userRepo user.Repository
	rdb      *redis.Client
}

func NewService(repo Repository, userRepo user.Repository, rdb *redis.Client) *Service {
	return &Service{
		repo:     repo,
		userRepo: userRepo,
		rdb:      rdb,
	}
}

func (s *Service) AddFriend(ctx context.Context, userID, friendID uuid.UUID) error {
	if userID == friendID {
		return ErrSelfRequest
	}

	existing, _ := s.repo.GetByUserPair(ctx, userID, friendID)
	if existing != nil {
		switch existing.Status {
		case domain.FriendAccepted:
			return ErrAlreadyFriends
		case domain.FriendBlocked:
			return ErrBlocked
		default:
			return nil
		}
	}

	if err := s.repo.AddRequest(ctx, userID, friendID); err != nil {
		return fmt.Errorf("add friend: %w", err)
	}

	log.Info().Str("from", userID.String()).Str("to", friendID.String()).Msg("friend request sent")
	return nil
}

func (s *Service) AcceptFriend(ctx context.Context, userID, requesterID uuid.UUID) error {
	f, err := s.repo.GetByUserPair(ctx, requesterID, userID)
	if err != nil {
		return ErrNotFound
	}
	if f.Status != domain.FriendPending {
		return errors.New("no pending request")
	}

	if err := s.repo.Accept(ctx, requesterID, userID); err != nil {
		return fmt.Errorf("accept: %w", err)
	}

	log.Info().Str("user", userID.String()).Str("friend", requesterID.String()).Msg("friend request accepted")
	return nil
}

func (s *Service) RejectFriend(ctx context.Context, userID, requesterID uuid.UUID) error {
	return s.repo.Reject(ctx, requesterID, userID)
}

func (s *Service) GetFriends(ctx context.Context, userID uuid.UUID) ([]FriendWithProfile, error) {
	friends, err := s.repo.GetFriends(ctx, userID)
	if err != nil {
		return nil, err
	}

	var result []FriendWithProfile
	for _, f := range friends {
		u, err := s.userRepo.GetByID(ctx, f.FriendID)
		if err != nil {
			continue
		}
		result = append(result, FriendWithProfile{
			UserID:      u.ID,
			DisplayName: u.DisplayName,
			AvatarURL:   avatarURL(u),
			Faction:     u.Faction,
			Level:       u.AccountLevel,
		})
	}
	return result, nil
}

func (s *Service) GetPending(ctx context.Context, userID uuid.UUID) ([]FriendWithProfile, error) {
	pending, err := s.repo.GetPending(ctx, userID)
	if err != nil {
		return nil, err
	}

	var result []FriendWithProfile
	for _, f := range pending {
		u, err := s.userRepo.GetByID(ctx, f.UserID)
		if err != nil {
			continue
		}
		result = append(result, FriendWithProfile{
			UserID:      u.ID,
			DisplayName: u.DisplayName,
			AvatarURL:   avatarURL(u),
			Faction:     u.Faction,
			Level:       u.AccountLevel,
		})
	}
	return result, nil
}

func (s *Service) GetFeed(ctx context.Context, userID uuid.UUID) ([]FeedEntry, error) {
	_, err := s.repo.GetFriendIDs(ctx, userID)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

type FriendWithProfile struct {
	UserID      uuid.UUID       `json:"user_id"`
	DisplayName string          `json:"display_name"`
	AvatarURL   string          `json:"avatar_url"`
	Faction     *domain.Faction `json:"faction"`
	Level       int             `json:"level"`
}

type FeedEntry struct {
	UserID          uuid.UUID `json:"user_id"`
	DisplayName     string    `json:"display_name"`
	RunID           uuid.UUID `json:"run_id"`
	DistanceM       float64   `json:"distance_m"`
	TerritoryPoints int       `json:"territory_points"`
	CreatedAt       string    `json:"created_at"`
}
