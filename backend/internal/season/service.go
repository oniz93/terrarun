package season

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/terrarun/backend/internal/domain"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetCurrentSeason(ctx context.Context) (*domain.Season, error) {
	season, err := s.repo.GetCurrent(ctx)
	if err != nil {
		return s.getOrCreateCurrent(ctx)
	}

	if time.Now().After(season.EndsAt) {
		return s.getOrCreateCurrent(ctx)
	}

	return season, nil
}

func (s *Service) getOrCreateCurrent(ctx context.Context) (*domain.Season, error) {
	season := &domain.Season{
		ID:       uuid.New(),
		Name:     fmt.Sprintf("Season %d", time.Now().Month()),
		StartsAt: time.Now(),
		EndsAt:   time.Now().Add(30 * 24 * time.Hour),
		IsActive: true,
	}

	if err := s.repo.Create(ctx, season); err != nil {
		return nil, fmt.Errorf("create season: %w", err)
	}

	return season, nil
}

func (s *Service) GetSeasonByID(ctx context.Context, id uuid.UUID) (*domain.Season, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) ListSeasons(ctx context.Context) ([]domain.Season, error) {
	return s.repo.List(ctx)
}

func (s *Service) JoinSeason(ctx context.Context, userID uuid.UUID, faction domain.Faction) error {
	season, err := s.GetCurrentSeason(ctx)
	if err != nil {
		return err
	}

	return s.repo.AddParticipant(ctx, userID, season.ID, faction)
}

func (s *Service) GetLeaderboard(ctx context.Context, seasonID uuid.UUID) ([]domain.SeasonParticipant, error) {
	return s.repo.GetParticipants(ctx, seasonID)
}

func (s *Service) UpdateScore(ctx context.Context, userID, seasonID uuid.UUID, points int) error {
	return s.repo.UpdateParticipantScore(ctx, userID, seasonID, points)
}
