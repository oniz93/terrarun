package bot

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/terrarun/backend/internal/domain"
)

type Repository interface {
	Create(ctx context.Context, bot *domain.Bot) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Bot, error)
	ListActive(ctx context.Context) ([]domain.Bot, error)
	ListByFaction(ctx context.Context, faction domain.Faction) ([]domain.Bot, error)
	CountActive(ctx context.Context) (int, error)
	IncrementRuns(ctx context.Context, id uuid.UUID) error
	Deactivate(ctx context.Context, id uuid.UUID) error
}

type PostgresRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

func scanBot(row pgx.Row) (*domain.Bot, error) {
	var b domain.Bot
	err := row.Scan(&b.ID, &b.DisplayName, &b.Faction, &b.AvatarSeed,
		&b.HomeLat, &b.HomeLng, &b.ActiveRadiusKM, &b.AvgDistanceM,
		&b.TotalRuns, &b.CreatedAt, &b.DeactivatedAt)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func scanBots(rows pgx.Rows) ([]domain.Bot, error) {
	defer rows.Close()
	var bots []domain.Bot
	for rows.Next() {
		var b domain.Bot
		err := rows.Scan(&b.ID, &b.DisplayName, &b.Faction, &b.AvatarSeed,
			&b.HomeLat, &b.HomeLng, &b.ActiveRadiusKM, &b.AvgDistanceM,
			&b.TotalRuns, &b.CreatedAt, &b.DeactivatedAt)
		if err != nil {
			return nil, err
		}
		bots = append(bots, b)
	}
	return bots, nil
}

func (r *PostgresRepo) Create(ctx context.Context, bot *domain.Bot) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO bots (id, display_name, faction, avatar_seed, home_lat, home_lng,
		 active_radius_km, avg_distance_m, total_runs, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		bot.ID, bot.DisplayName, bot.Faction, bot.AvatarSeed,
		bot.HomeLat, bot.HomeLng, bot.ActiveRadiusKM, bot.AvgDistanceM,
		bot.TotalRuns, bot.CreatedAt)
	return err
}

func (r *PostgresRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Bot, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, display_name, faction, avatar_seed, home_lat, home_lng,
		 active_radius_km, avg_distance_m, total_runs, created_at, deactivated_at
		 FROM bots WHERE id = $1`, id)
	return scanBot(row)
}

func (r *PostgresRepo) ListActive(ctx context.Context) ([]domain.Bot, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, display_name, faction, avatar_seed, home_lat, home_lng,
		 active_radius_km, avg_distance_m, total_runs, created_at, deactivated_at
		 FROM bots WHERE deactivated_at IS NULL`)
	if err != nil {
		return nil, err
	}
	return scanBots(rows)
}

func (r *PostgresRepo) ListByFaction(ctx context.Context, faction domain.Faction) ([]domain.Bot, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, display_name, faction, avatar_seed, home_lat, home_lng,
		 active_radius_km, avg_distance_m, total_runs, created_at, deactivated_at
		 FROM bots WHERE deactivated_at IS NULL AND faction = $1`, faction)
	if err != nil {
		return nil, err
	}
	return scanBots(rows)
}

func (r *PostgresRepo) CountActive(ctx context.Context) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM bots WHERE deactivated_at IS NULL`).Scan(&count)
	return count, err
}

func (r *PostgresRepo) IncrementRuns(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE bots SET total_runs = total_runs + 1 WHERE id = $1`, id)
	return err
}

func (r *PostgresRepo) Deactivate(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE bots SET deactivated_at = NOW() WHERE id = $1`, id)
	return err
}
