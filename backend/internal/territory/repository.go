package territory

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/terrarun/backend/internal/domain"
)

type Repository interface {
	GetHex(ctx context.Context, h3Index int64) (*domain.Hex, error)
	UpsertHex(ctx context.Context, hex *domain.Hex) error
	GetHexesInBounds(ctx context.Context, swLat, swLng, neLat, neLng float64) ([]domain.Hex, error)
	BatchUpsertHexes(ctx context.Context, hexes []domain.Hex) error
	BatchDecayHP(ctx context.Context, limit int) (int, error)
	RecordChange(ctx context.Context, change *domain.TerritoryChange) error
	GetRecentChanges(ctx context.Context, since time.Time) ([]domain.TerritoryChange, error)
	GetRegionStats(ctx context.Context) (*domain.TerritoryStats, error)
	GetUserCapturedHexes(ctx context.Context, userID, runID uuid.UUID) ([]domain.Hex, error)
	CountClaimed(ctx context.Context) (int64, error)
}

type PostgresRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

func (r *PostgresRepo) GetHex(ctx context.Context, h3Index int64) (*domain.Hex, error) {
	var h domain.Hex
	var ownedBy *string
	err := r.pool.QueryRow(ctx,
		`SELECT h3_index, owned_by, hp, captured_by, captured_at, last_decayed_at, is_natural
		 FROM hexes WHERE h3_index = $1`, h3Index,
	).Scan(&h.H3Index, &ownedBy, &h.HP, &h.CapturedByID, &h.CapturedAt, &h.LastDecayedAt, &h.IsNatural)
	if err != nil {
		return nil, err
	}
	if ownedBy != nil {
		f := domain.Faction(*ownedBy)
		h.OwnedBy = &f
	}
	return &h, nil
}

func (r *PostgresRepo) UpsertHex(ctx context.Context, hex *domain.Hex) error {
	var ownedBy *string
	if hex.OwnedBy != nil {
		s := string(*hex.OwnedBy)
		ownedBy = &s
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO hexes (h3_index, owned_by, hp, captured_by, captured_at, last_decayed_at, is_natural, lat, lng)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 ON CONFLICT (h3_index) DO UPDATE SET
		     owned_by = EXCLUDED.owned_by,
		     hp = EXCLUDED.hp,
		     captured_by = CASE WHEN EXCLUDED.owned_by IS NOT NULL THEN EXCLUDED.captured_by ELSE hexes.captured_by END,
		     captured_at = CASE WHEN EXCLUDED.owned_by IS NOT NULL THEN EXCLUDED.captured_at ELSE hexes.captured_at END,
		     is_natural = CASE WHEN $7 THEN true ELSE hexes.is_natural END,
		     lat = EXCLUDED.lat,
		     lng = EXCLUDED.lng`,
		hex.H3Index, ownedBy, hex.HP, hex.CapturedByID, hex.CapturedAt, hex.LastDecayedAt, hex.IsNatural, hex.Lat, hex.Lng)
	return err
}

func (r *PostgresRepo) GetHexesInBounds(ctx context.Context, swLat, swLng, neLat, neLng float64) ([]domain.Hex, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT h3_index, owned_by, hp, captured_by, captured_at, last_decayed_at, is_natural
		 FROM hexes 
		 WHERE owned_by IS NOT NULL
		   AND lat BETWEEN $1 AND $3
		   AND lng BETWEEN $2 AND $4
		 LIMIT 50000`, swLat, swLng, neLat, neLng)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hexes []domain.Hex
	for rows.Next() {
		var h domain.Hex
		var ownedBy *string
		if err := rows.Scan(&h.H3Index, &ownedBy, &h.HP, &h.CapturedByID, &h.CapturedAt, &h.LastDecayedAt, &h.IsNatural); err != nil {
			return nil, err
		}
		if ownedBy != nil {
			f := domain.Faction(*ownedBy)
			h.OwnedBy = &f
		}
		hexes = append(hexes, h)
	}
	return hexes, nil
}

func (r *PostgresRepo) BatchUpsertHexes(ctx context.Context, hexes []domain.Hex) error {
	batch := &pgx.Batch{}
	for _, h := range hexes {
		var ownedBy *string
		if h.OwnedBy != nil {
			s := string(*h.OwnedBy)
			ownedBy = &s
		}
		batch.Queue(
			`INSERT INTO hexes (h3_index, owned_by, hp, captured_by, captured_at, last_decayed_at, is_natural, lat, lng)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			 ON CONFLICT (h3_index) DO UPDATE SET
			     owned_by = EXCLUDED.owned_by, hp = EXCLUDED.hp,
			     captured_by = EXCLUDED.captured_by, captured_at = EXCLUDED.captured_at,
			     lat = EXCLUDED.lat, lng = EXCLUDED.lng`,
			h.H3Index, ownedBy, h.HP, h.CapturedByID, h.CapturedAt, h.LastDecayedAt, h.IsNatural, h.Lat, h.Lng)
	}
	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()
	for range hexes {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}
	return nil
}

func (r *PostgresRepo) BatchDecayHP(ctx context.Context, limit int) (int, error) {
	tag, err := r.pool.Exec(ctx,
		`UPDATE hexes SET hp = hp - 1, last_decayed_at = NOW()
		 WHERE h3_index IN (
		     SELECT h3_index FROM hexes WHERE owned_by IS NOT NULL AND hp > 1
		     LIMIT $1
		 )`, limit)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

func (r *PostgresRepo) RecordChange(ctx context.Context, change *domain.TerritoryChange) error {
	var prevOwner, newOwner *string
	if change.PreviousOwner != nil {
		s := string(*change.PreviousOwner)
		prevOwner = &s
	}
	if change.NewOwner != nil {
		s := string(*change.NewOwner)
		newOwner = &s
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO territory_changes (h3_index, run_id, previous_owner, new_owner, hp_before, hp_after, changed_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		change.H3Index, change.RunID, prevOwner, newOwner, change.HPBefore, change.HPAfter, change.ChangedAt)
	return err
}

func (r *PostgresRepo) GetRecentChanges(ctx context.Context, since time.Time) ([]domain.TerritoryChange, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, h3_index, run_id, previous_owner, new_owner, hp_before, hp_after, changed_at
		 FROM territory_changes WHERE changed_at >= $1 ORDER BY changed_at DESC LIMIT 1000`, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var changes []domain.TerritoryChange
	for rows.Next() {
		var c domain.TerritoryChange
		var prevOwner, newOwner *string
		if err := rows.Scan(&c.ID, &c.H3Index, &c.RunID, &prevOwner, &newOwner, &c.HPBefore, &c.HPAfter, &c.ChangedAt); err != nil {
			return nil, err
		}
		if prevOwner != nil {
			f := domain.Faction(*prevOwner)
			c.PreviousOwner = &f
		}
		if newOwner != nil {
			f := domain.Faction(*newOwner)
			c.NewOwner = &f
		}
		changes = append(changes, c)
	}
	return changes, nil
}

func (r *PostgresRepo) GetRegionStats(ctx context.Context) (*domain.TerritoryStats, error) {
	var stats domain.TerritoryStats
	row := r.pool.QueryRow(ctx,
		`SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE owned_by IS NOT NULL) as claimed,
			COUNT(*) FILTER (WHERE owned_by = 'neon') as neon,
			COUNT(*) FILTER (WHERE owned_by = 'umbra') as umbra
		 FROM hexes`)
	err := row.Scan(&stats.TotalHexes, &stats.ClaimedHexes, &stats.NeonHexes, &stats.UmbraHexes)
	if err != nil {
		return nil, err
	}
	if stats.ClaimedHexes > 0 {
		stats.NeonPercentage = float64(stats.NeonHexes) / float64(stats.ClaimedHexes) * 100
		stats.UmbraPercentage = float64(stats.UmbraHexes) / float64(stats.ClaimedHexes) * 100
	}

	var contested int64
	r.pool.QueryRow(ctx,
		`SELECT COUNT(DISTINCT h3_index) FROM territory_changes WHERE changed_at >= NOW() - INTERVAL '24 hours'`,
	).Scan(&contested)
	stats.ContestedZones = contested

	return &stats, nil
}

func (r *PostgresRepo) GetUserCapturedHexes(ctx context.Context, userID, runID uuid.UUID) ([]domain.Hex, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT h.h3_index, h.owned_by, h.hp, h.captured_by, h.captured_at, h.last_decayed_at, h.is_natural
		 FROM hexes h
		 JOIN territory_changes tc ON tc.h3_index = h.h3_index
		 WHERE tc.run_id = $1`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hexes []domain.Hex
	for rows.Next() {
		var h domain.Hex
		var ownedBy *string
		if err := rows.Scan(&h.H3Index, &ownedBy, &h.HP, &h.CapturedByID, &h.CapturedAt, &h.LastDecayedAt, &h.IsNatural); err != nil {
			return nil, err
		}
		if ownedBy != nil {
			f := domain.Faction(*ownedBy)
			h.OwnedBy = &f
		}
		hexes = append(hexes, h)
	}
	return hexes, nil
}

func (r *PostgresRepo) CountClaimed(ctx context.Context) (int64, error) {
	var count int64
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM hexes WHERE owned_by IS NOT NULL`).Scan(&count)
	return count, err
}
