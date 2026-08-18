package bot

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	"github.com/terrarun/backend/internal/domain"
	"github.com/terrarun/backend/internal/territory"
	"github.com/terrarun/backend/pkg/h3util"
)

type Service struct {
	repo          Repository
	db            *pgxpool.Pool
	rdb           *redis.Client
	territoryRepo territory.Repository
}

func NewService(repo Repository, db *pgxpool.Pool, rdb *redis.Client, tRepo territory.Repository) *Service {
	return &Service{
		repo:          repo,
		db:            db,
		rdb:           rdb,
		territoryRepo: tRepo,
	}
}

type SeederConfig struct {
	CenterLat      float64
	CenterLng      float64
	RadiusKM       float64
	BotCount       int
	Faction        domain.Faction
	TerritoryRatio float64
}

func (s *Service) SeedOnboarding(ctx context.Context, cfg SeederConfig) error {
	existing, err := s.repo.CountActive(ctx)
	if err != nil {
		return err
	}
	if existing > 0 {
		log.Info().Int("existing", existing).Msg("bots already seeded, skipping")
		return nil
	}

	for i := 0; i < cfg.BotCount; i++ {
		angle := 2 * math.Pi * float64(i) / float64(cfg.BotCount)
		distKM := cfg.RadiusKM * (0.3 + rand.Float64()*0.7)
		lat := cfg.CenterLat + distKM*math.Cos(angle)/111.0
		lng := cfg.CenterLng + distKM*math.Sin(angle)/(111.0*math.Cos(cfg.CenterLat*math.Pi/180))

		avatarSeed := fmt.Sprintf("bot_%s", uuid.New().String()[:8])

		bot := &domain.Bot{
			ID:             uuid.New(),
			DisplayName:    fmt.Sprintf("Runner_%s", uuid.New().String()[:6]),
			Faction:        cfg.Faction,
			AvatarSeed:     &avatarSeed,
			HomeLat:        lat,
			HomeLng:        lng,
			ActiveRadiusKM: 5,
			AvgDistanceM:   3000 + rand.Float64()*4000,
			TotalRuns:      0,
			CreatedAt:      time.Now(),
		}

		if err := s.repo.Create(ctx, bot); err != nil {
			log.Warn().Err(err).Msg("failed to create onboarding bot")
			continue
		}
	}

	log.Info().Int("count", cfg.BotCount).Str("faction", string(cfg.Faction)).Msg("bots seeded")
	return nil
}

func (s *Service) ShouldSpawnBots(ctx context.Context) bool {
	var realUsers int
	s.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM users WHERE email NOT LIKE '%@terrarun.game'`).Scan(&realUsers)
	return realUsers < 20
}

func (s *Service) MaintainBotBalance(ctx context.Context) error {
	activeBots, err := s.repo.CountActive(ctx)
	if err != nil {
		return err
	}

	var realUsers int
	s.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM users WHERE email NOT LIKE '%@terrarun.game'`).Scan(&realUsers)

	targetPerFaction := int(math.Max(5, float64(10-realUsers)))
	needed := targetPerFaction*2 - activeBots
	if needed <= 0 {
		return nil
	}

	spawnFaction := domain.FactionNeon
	centerLat, centerLng := s.spawnCenter(ctx)
	for i := 0; i < needed; i++ {
		avatarSeed := fmt.Sprintf("abot_%s", uuid.New().String()[:8])
		lat := centerLat + (rand.Float64()-0.5)*0.1
		lng := centerLng + (rand.Float64()-0.5)*0.1

		bot := &domain.Bot{
			ID:             uuid.New(),
			DisplayName:    fmt.Sprintf("BotRunner_%s", uuid.New().String()[:6]),
			Faction:        spawnFaction,
			AvatarSeed:     &avatarSeed,
			HomeLat:        lat,
			HomeLng:        lng,
			ActiveRadiusKM: 3 + rand.Float64()*4,
			AvgDistanceM:   2000 + rand.Float64()*5000,
			TotalRuns:      0,
			CreatedAt:      time.Now(),
		}

		if err := s.repo.Create(ctx, bot); err != nil {
			log.Warn().Err(err).Msg("failed to spawn bot")
			continue
		}

		if spawnFaction == domain.FactionNeon {
			spawnFaction = domain.FactionUmbra
		} else {
			spawnFaction = domain.FactionNeon
		}
	}

	log.Info().Int("spawned", needed).Msg("active bots spawned")
	return nil
}

func (s *Service) SimulateBotRun(ctx context.Context, bot domain.Bot) error {
	hour := time.Now().Hour()
	if hour < 6 || hour >= 22 {
		return nil
	}

	heading := rand.Float64() * 360
	sinH := math.Sin(heading * math.Pi / 180)
	cosH := math.Cos(heading * math.Pi / 180)
	steps := int(bot.AvgDistanceM / 10)
	if steps < 10 {
		steps = 10
	}

	factionStr := string(bot.Faction)
	for step := 0; step < steps; step++ {
		frac := float64(step) / float64(steps)
		lat := bot.HomeLat + frac*bot.AvgDistanceM*sinH/111000.0
		lng := bot.HomeLng + frac*bot.AvgDistanceM*cosH/(111000.0*math.Cos(bot.HomeLat*math.Pi/180))

		cell, err := h3util.LatLngToCell(lat, lng, 11)
		if err != nil {
			continue
		}
		cellInt := int64(cell)

		var currentOwner *string
		var currentHP int
		err = s.db.QueryRow(ctx,
			`SELECT owned_by, hp FROM hexes WHERE h3_index = $1`, cellInt).Scan(&currentOwner, &currentHP)
		if err != nil && err != pgx.ErrNoRows {
			continue
		}
		if err == pgx.ErrNoRows {
			// Neutral hex: treat as unowned with 0 HP.
			currentOwner = nil
			currentHP = 0
		}

		if currentOwner != nil && *currentOwner == factionStr {
			continue
		}

		hp := rand.Intn(3) + 1
		s.db.Exec(ctx,
			`INSERT INTO hexes (h3_index, owned_by, hp, captured_at, last_decayed_at)
			 VALUES ($1, $2, $3, NOW(), NOW())
			 ON CONFLICT (h3_index) DO UPDATE SET owned_by = $2, hp = $3, captured_at = NOW()`,
			cellInt, factionStr, hp)
		s.db.Exec(ctx,
			`INSERT INTO territory_changes (h3_index, previous_owner, new_owner, hp_before, hp_after)
			 VALUES ($1, $2, $3, $4, $5)`,
			cellInt, currentOwner, factionStr, currentHP, hp)
	}

	return s.repo.IncrementRuns(ctx, bot.ID)
}

// spawnCenter returns the centroid of existing bots' homes, falling back to the
// Vancouver seeding area so new bots stay inside the seeded game world.
func (s *Service) spawnCenter(ctx context.Context) (float64, float64) {
	bots, err := s.repo.ListActive(ctx)
	if err == nil && len(bots) > 0 {
		var sumLat, sumLng float64
		for _, b := range bots {
			sumLat += b.HomeLat
			sumLng += b.HomeLng
		}
		return sumLat / float64(len(bots)), sumLng / float64(len(bots))
	}
	// Vancouver (matches cmd/seed_vancouver_v6).
	return 49.2253, -123.0050
}

func (s *Service) GetActiveBots(ctx context.Context) ([]domain.Bot, error) {
	return s.repo.ListActive(ctx)
}

func (s *Service) ScheduleRuns(ctx context.Context) {
	bots, err := s.repo.ListActive(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to list active bots")
		return
	}

	for _, bot := range bots {
		if err := s.SimulateBotRun(ctx, bot); err != nil {
			log.Warn().Err(err).Str("bot_id", bot.ID.String()).Msg("bot run simulation failed")
		}
	}
}
