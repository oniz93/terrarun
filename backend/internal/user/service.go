package user

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	"github.com/terrarun/backend/internal/domain"
	"github.com/terrarun/backend/internal/storage"
)

var (
	ErrNotFound = errors.New("user not found")
)

type Service struct {
	repo         Repository
	rdb          *redis.Client
	fileStore    *storage.RustFS
	avatarBucket string
}

func NewService(repo Repository, rdb *redis.Client, fileStore *storage.RustFS) *Service {
	return &Service{
		repo:         repo,
		rdb:          rdb,
		fileStore:    fileStore,
		avatarBucket: "avatars",
	}
}

func (s *Service) GetMe(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	return s.repo.GetByID(ctx, userID)
}

func (s *Service) UpdateMe(ctx context.Context, userID uuid.UUID, req UpdateRequest) (*domain.User, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, ErrNotFound
	}
	if req.DisplayName != nil {
		user.DisplayName = *req.DisplayName
	}
	if req.PhoneHash != nil {
		user.PhoneHash = req.PhoneHash
	}
	if err := s.repo.Update(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *Service) UploadAvatar(ctx context.Context, userID uuid.UUID, file multipart.File, header *multipart.FileHeader) (string, error) {
	if header.Size > 5<<20 {
		return "", fmt.Errorf("file too large: max 5MB")
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}
	key := fmt.Sprintf("%s/%s", userID.String(), header.Filename)
	url, err := s.fileStore.Upload(ctx, s.avatarBucket, key, data)
	if err != nil {
		return "", fmt.Errorf("upload avatar: %w", err)
	}
	if err := s.repo.Update(ctx, &domain.User{ID: userID, AvatarURL: &url}); err != nil {
		return "", err
	}
	log.Info().Str("user_id", userID.String()).Str("avatar", url).Msg("avatar uploaded")
	return url, nil
}

func (s *Service) GetPublic(ctx context.Context, userID uuid.UUID) (*domain.PublicUser, error) {
	u, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &domain.PublicUser{
		ID:           u.ID,
		DisplayName:  u.DisplayName,
		Faction:      u.Faction,
		AvatarURL:    u.AvatarURL,
		AccountLevel: u.AccountLevel,
		RunnerTier:   u.RunnerTier,
		CreatedAt:    u.CreatedAt,
	}, nil
}

func (s *Service) MatchContacts(ctx context.Context, hashes []string) (*ContactsMatchResult, error) {
	var matches []ContactMatch
	neonCount, umbraCount := 0, 0

	for _, hash := range hashes {
		user, err := s.repo.GetByPhoneHash(ctx, hash)
		if err != nil {
			continue
		}
		var f string
		if user.Faction != nil {
			f = string(*user.Faction)
		}
		matches = append(matches, ContactMatch{
			Hash:        hash,
			UserID:      user.ID,
			DisplayName: user.DisplayName,
			Faction:     f,
		})
		if user.Faction != nil {
			switch *user.Faction {
			case domain.FactionNeon:
				neonCount++
			case domain.FactionUmbra:
				umbraCount++
			}
		}
	}

	return &ContactsMatchResult{
		Matches:       matches,
		FactionCounts: map[string]int{"neon": neonCount, "umbra": umbraCount},
	}, nil
}

func (s *Service) ExportData(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	return s.repo.Export(ctx, userID)
}

func (s *Service) DeleteAccount(ctx context.Context, userID uuid.UUID) error {
	return s.repo.SoftDelete(ctx, userID)
}

type UpdateRequest struct {
	DisplayName *string `json:"display_name,omitempty"`
	PhoneHash   *string `json:"phone_hash,omitempty"`
}

type ContactMatch struct {
	Hash        string    `json:"hash"`
	UserID      uuid.UUID `json:"user_id"`
	DisplayName string    `json:"display_name"`
	Faction     string    `json:"faction"`
}

type ContactsMatchResult struct {
	Matches       []ContactMatch `json:"matches"`
	FactionCounts map[string]int `json:"faction_counts"`
}
