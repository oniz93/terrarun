package territory

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/terrarun/backend/internal/domain"
	"github.com/uber/h3-go/v4"
)

type SeedingService struct {
	repo Repository
	pool *pgxpool.Pool
}

func NewSeedingService(repo Repository, pool *pgxpool.Pool) *SeedingService {
	return &SeedingService{
		repo: repo,
		pool: pool,
	}
}

func (s *SeedingService) SeedHex(ctx context.Context, cell h3.Cell, faction string) (bool, error) {
	latLng, err := h3.CellToLatLng(cell)
	if err != nil {
		return false, err
	}
	lat, lng := latLng.Lat, latLng.Lng

	onLand, err := s.isOnLand(ctx, lng, lat)
	if err != nil || !onLand {
		return false, err
	}

	isWater, err := s.isWater(ctx, lng, lat)
	if err != nil || isWater {
		return false, err
	}

	f := domain.Faction(faction)
	err = s.repo.UpsertHex(ctx, &domain.Hex{
		H3Index:       int64(cell),
		OwnedBy:       &f,
		HP:            10,
		CapturedAt:    time.Now(),
		LastDecayedAt: time.Now(),
		IsNatural:     true,
		Lat:           lat,
		Lng:           lng,
	})
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *SeedingService) SeedHexWithFeatures(ctx context.Context, cell h3.Cell, faction string) (bool, []string, error) {
	latLng, err := h3.CellToLatLng(cell)
	if err != nil {
		return false, nil, err
	}
	lat, lng := latLng.Lat, latLng.Lng

	features, err := s.classifyPoint(ctx, lng, lat)
	if err != nil {
		return false, nil, err
	}

	if features.IsWater || !features.IsLand {
		return false, features.Slice(), nil
	}

	f := domain.Faction(faction)
	err = s.repo.UpsertHex(ctx, &domain.Hex{
		H3Index:       int64(cell),
		OwnedBy:       &f,
		HP:            10,
		CapturedAt:    time.Now(),
		LastDecayedAt: time.Now(),
		IsNatural:     features.IsProtected || features.IsForest || features.IsMountain,
		Lat:           lat,
		Lng:           lng,
	})
	if err != nil {
		return false, features.Slice(), err
	}
	return true, features.Slice(), nil
}

func (s *SeedingService) isOnLand(ctx context.Context, lng, lat float64) (bool, error) {
	var onLand bool
	err := s.pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM global_landmask WHERE ST_Intersects(geom, ST_SetSRID(ST_MakePoint($1, $2), 4326)))",
		lng, lat,
	).Scan(&onLand)
	if err != nil {
		return false, nil
	}
	return onLand, nil
}

func (s *SeedingService) isWater(ctx context.Context, lng, lat float64) (bool, error) {
	var isWater bool
	err := s.pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM osm_water WHERE ST_Intersects(geom, ST_SetSRID(ST_MakePoint($1, $2), 4326)))",
		lng, lat,
	).Scan(&isWater)
	if err != nil {
		return false, nil
	}
	return isWater, nil
}

type PointFeatures struct {
	IsLand      bool
	IsWater     bool
	IsForest    bool
	IsMountain  bool
	IsProtected bool
}

func (pf PointFeatures) Slice() []string {
	var features []string
	if pf.IsLand {
		features = append(features, "land")
	}
	if pf.IsWater {
		features = append(features, "water")
	}
	if pf.IsForest {
		features = append(features, "forest")
	}
	if pf.IsMountain {
		features = append(features, "mountain")
	}
	if pf.IsProtected {
		features = append(features, "protected")
	}
	return features
}

func (s *SeedingService) classifyPoint(ctx context.Context, lng, lat float64) (PointFeatures, error) {
	var pf PointFeatures

	err := s.pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM global_landmask WHERE ST_Intersects(geom, ST_SetSRID(ST_MakePoint($1, $2), 4326)))",
		lng, lat,
	).Scan(&pf.IsLand)
	if err != nil {
		return pf, err
	}

	err = s.pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM osm_water WHERE ST_Intersects(geom, ST_SetSRID(ST_MakePoint($1, $2), 4326)))",
		lng, lat,
	).Scan(&pf.IsWater)
	if err != nil {
		return pf, err
	}

	err = s.pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM osm_forest WHERE ST_Intersects(geom, ST_SetSRID(ST_MakePoint($1, $2), 4326)))",
		lng, lat,
	).Scan(&pf.IsForest)
	if err != nil {
		pf.IsForest = false
	}

	err = s.pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM osm_mountain WHERE ST_Intersects(geom, ST_SetSRID(ST_MakePoint($1, $2), 4326)))",
		lng, lat,
	).Scan(&pf.IsMountain)
	if err != nil {
		pf.IsMountain = false
	}

	err = s.pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM osm_protected WHERE ST_Intersects(geom, ST_SetSRID(ST_MakePoint($1, $2), 4326)))",
		lng, lat,
	).Scan(&pf.IsProtected)
	if err != nil {
		pf.IsProtected = false
	}

	return pf, nil
}
