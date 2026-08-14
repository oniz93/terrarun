package season

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/terrarun/backend/internal/domain"
)

type Repository interface {
	Create(ctx context.Context, s *domain.Season) error
	DeactivateAll(ctx context.Context) error
	GetCurrent(ctx context.Context) (*domain.Season, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Season, error)
	List(ctx context.Context) ([]domain.Season, error)
	AddParticipant(ctx context.Context, userID, seasonID uuid.UUID) error
	UpdateParticipantScore(ctx context.Context, userID, seasonID uuid.UUID, points int) error
	GetParticipants(ctx context.Context, seasonID uuid.UUID) ([]domain.SeasonParticipant, error)
}

type PostgresRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

const defaultTiersConfig = `{"rookie":0,"runner":1000,"sprinter":5000,"elite":15000,"legend":30000}`

func (r *PostgresRepo) Create(ctx context.Context, s *domain.Season) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO seasons (id, name, start_date, end_date, tiers_config, is_active, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, NOW())`,
		s.ID, s.Name, s.StartDate, s.EndDate, defaultTiersConfig, s.IsActive)
	return err
}

func (r *PostgresRepo) DeactivateAll(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `UPDATE seasons SET is_active = false WHERE is_active = true`)
	return err
}

func (r *PostgresRepo) GetCurrent(ctx context.Context) (*domain.Season, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, name, start_date, end_date, is_active, created_at
		 FROM seasons WHERE is_active = true ORDER BY start_date DESC LIMIT 1`)
	var s domain.Season
	err := row.Scan(&s.ID, &s.Name, &s.StartDate, &s.EndDate, &s.IsActive, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *PostgresRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Season, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, name, start_date, end_date, is_active, created_at
		 FROM seasons WHERE id = $1`, id)
	var s domain.Season
	err := row.Scan(&s.ID, &s.Name, &s.StartDate, &s.EndDate, &s.IsActive, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *PostgresRepo) List(ctx context.Context) ([]domain.Season, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, start_date, end_date, is_active, created_at
		 FROM seasons ORDER BY start_date DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var seasons []domain.Season
	for rows.Next() {
		var s domain.Season
		if err := rows.Scan(&s.ID, &s.Name, &s.StartDate, &s.EndDate, &s.IsActive, &s.CreatedAt); err != nil {
			return nil, err
		}
		seasons = append(seasons, s)
	}
	return seasons, nil
}

func (r *PostgresRepo) AddParticipant(ctx context.Context, userID, seasonID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO season_participants (user_id, season_id, runner_points, territory_points, current_tier)
		 VALUES ($1, $2, 0, 0, 'rookie')
		 ON CONFLICT (user_id, season_id) DO NOTHING`,
		userID, seasonID)
	return err
}

func (r *PostgresRepo) UpdateParticipantScore(ctx context.Context, userID, seasonID uuid.UUID, points int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE season_participants SET runner_points = runner_points + $1
		 WHERE user_id = $2 AND season_id = $3`,
		points, userID, seasonID)
	return err
}

func (r *PostgresRepo) GetParticipants(ctx context.Context, seasonID uuid.UUID) ([]domain.SeasonParticipant, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT sp.user_id, sp.season_id, u.faction, sp.runner_points, sp.territory_points, sp.current_tier, u.display_name
		 FROM season_participants sp
		 JOIN users u ON u.id = sp.user_id
		 WHERE sp.season_id = $1
		 ORDER BY sp.runner_points DESC`, seasonID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var participants []domain.SeasonParticipant
	for rows.Next() {
		var p domain.SeasonParticipant
		if err := rows.Scan(&p.UserID, &p.SeasonID, &p.Faction, &p.RunnerPoints, &p.TerritoryPoints, &p.CurrentTier, &p.DisplayName); err != nil {
			return nil, err
		}
		participants = append(participants, p)
	}
	return participants, nil
}
