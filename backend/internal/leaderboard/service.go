package leaderboard

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type Service struct {
	repo *RedisRepo
}

func NewService(repo *RedisRepo) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetTeamLeaderboard(ctx context.Context, region string) ([]redis.Z, error) {
	return s.repo.GetTeamLeaderboard(ctx, region, 100)
}

func (s *Service) GetRunnerLeaderboard(ctx context.Context, seasonID string) ([]redis.Z, error) {
	return s.repo.GetRunnerLeaderboard(ctx, seasonID, 100)
}

func (s *Service) GetFriendsLeaderboard(ctx context.Context, userID string) ([]redis.Z, error) {
	return nil, nil
}

func (s *Service) UpdateScores(ctx context.Context, userID, faction, seasonID string, runnerPoints, territoryPoints int64) error {
	if err := s.repo.UpdateRunnerScore(ctx, seasonID, userID, runnerPoints); err != nil {
		return fmt.Errorf("update runner score: %w", err)
	}
	if err := s.repo.UpdateTeamScore(ctx, "global", faction, territoryPoints); err != nil {
		return fmt.Errorf("update team score: %w", err)
	}
	return nil
}
