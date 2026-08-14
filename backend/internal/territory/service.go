package territory

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/terrarun/backend/internal/config"
	"github.com/terrarun/backend/internal/domain"
)

type Service struct {
	repo  Repository
	redis *redis.Client
	cfg   *config.Config
}

func NewService(repo Repository, redis *redis.Client, cfg *config.Config) *Service {
	return &Service{
		repo:  repo,
		redis: redis,
		cfg:   cfg,
	}
}

func (s *Service) GetViewportHexes(ctx context.Context, swLat, swLng, neLat, neLng float64) ([]domain.Hex, error) {
	hexes, err := s.repo.GetHexesInBounds(ctx, swLat, swLng, neLat, neLng)
	if err != nil {
		return nil, fmt.Errorf("get hexes in bounds: %w", err)
	}
	return hexes, nil
}

func (s *Service) GetRegionStats(ctx context.Context) (*domain.TerritoryStats, error) {
	return s.repo.GetRegionStats(ctx)
}

func (s *Service) DecayAllHexes(ctx context.Context, batchSize int) (int, error) {
	total := 0
	for {
		affected, err := s.repo.BatchDecayHP(ctx, batchSize)
		if err != nil {
			return total, fmt.Errorf("decay hp: %w", err)
		}
		total += affected
		if affected < batchSize {
			break
		}
	}
	return total, nil
}

func (s *Service) CreateHex(ctx context.Context, hex *domain.Hex) error {
	return s.repo.UpsertHex(ctx, hex)
}
