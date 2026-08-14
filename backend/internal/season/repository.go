package season

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/terrarun/backend/internal/domain"
)

type Repository interface {
	Create(ctx context.Context, s *domain.Season) error
	GetCurrent(ctx context.Context) (*domain.Season, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Season, error)
	List(ctx context.Context) ([]domain.Season, error)
	AddParticipant(ctx context.Context, userID, seasonID uuid.UUID, faction domain.Faction) error
	UpdateParticipantScore(ctx context.Context, userID, seasonID uuid.UUID, points int) error
	GetParticipants(ctx context.Context, seasonID uuid.UUID) ([]domain.SeasonParticipant, error)
}

type PostgresRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

func (r *PostgresRepo) Create(ctx context.Context, s *domain.Season) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO seasons (id, name, starts_at, ends_at, is_active, created_at)
		 VALUES ($1, $2, $3, $4, $5, NOW())`,
		s.ID, s.Name, s.StartsAt, s.EndsAt, s.IsActive)
	return err
}

func (r *PostgresRepo) GetCurrent(ctx context.Context) (*domain.Season, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, name, starts_at, ends_at, is_active, created_at, updated_at
		 FROM seasons WHERE is_active = true ORDER BY starts_at DESC LIMIT 1`)
	var s domain.Season
	err := row.Scan(&s.ID, &s.Name, &s.StartsAt, &s.EndsAt, &s.IsActive, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *PostgresRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Season, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, name, starts_at, ends_at, is_active, created_at, updated_at
		 FROM seasons WHERE id = $1`, id)
	var s domain.Season
	err := row.Scan(&s.ID, &s.Name, &s.StartsAt, &s.EndsAt, &s.IsActive, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *PostgresRepo) List(ctx context.Context) ([]domain.Season, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, starts_at, ends_at, is_active, created_at, updated_at
		 FROM seasons ORDER BY starts_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var seasons []domain.Season
	for rows.Next() {
		var s domain.Season
		if err := rows.Scan(&s.ID, &s.Name, &s.StartsAt, &s.EndsAt, &s.IsActive, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		seasons = append(seasons, s)
	}
	return seasons, nil
}

func (r *PostgresRepo) AddParticipant(ctx context.Context, userID, seasonID uuid.UUID, faction domain.Faction) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO season_participants (user_id, season_id, faction, points)
		 VALUES ($1, $2, $3, 0)
		 ON CONFLICT (user_id, season_id) DO NOTHING`,
		userID, seasonID, faction)
	return err
}

func (r *PostgresRepo) UpdateParticipantScore(ctx context.Context, userID, seasonID uuid.UUID, points int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE season_participants SET points = points + $1, updated_at = NOW()
		 WHERE user_id = $2 AND season_id = $3`,
		points, userID, seasonID)
	return err
}

func (r *PostgresRepo) GetParticipants(ctx context.Context, seasonID uuid.UUID) ([]domain.SeasonParticipant, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT sp.user_id, sp.season_id, sp.faction, sp.points, u.display_name
		 FROM season_participants sp
		 JOIN users u ON u.id = sp.user_id
		 WHERE sp.season_id = $1
		 ORDER BY sp.points DESC`, seasonID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var participants []domain.SeasonParticipant
	for rows.Next() {
		var p domain.SeasonParticipant
		if err := rows.Scan(&p.UserID, &p.SeasonID, &p.Faction, &p.Points, &p.DisplayName); err != nil {
			return nil, err
		}
		participants = append(participants, p)
	}
	return participants, nil
}
