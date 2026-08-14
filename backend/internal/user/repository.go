package user

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/terrarun/backend/internal/domain"
)

type Repository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByGoogleID(ctx context.Context, googleID string) (*domain.User, error)
	GetByAppleID(ctx context.Context, appleID string) (*domain.User, error)
	GetByPhoneHash(ctx context.Context, hash string) (*domain.User, error)
	Create(ctx context.Context, user *domain.User) error
	Update(ctx context.Context, user *domain.User) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	Export(ctx context.Context, id uuid.UUID) (*domain.User, error)
	IncrementXP(ctx context.Context, id uuid.UUID, xp int64) error
	UpdateFaction(ctx context.Context, id uuid.UUID, faction domain.Faction) error
}

type PostgresRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

func scanUser(row pgx.Row) (*domain.User, error) {
	var u domain.User
	err := row.Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.PhoneHash,
		&u.AvatarURL, &u.Faction, &u.AccountLevel, &u.AccountXP,
		&u.GoogleID, &u.AppleID, &u.RunnerTier,
		&u.CreatedAt, &u.UpdatedAt, &u.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

const userCols = `id, email, password_hash, display_name, phone_hash, avatar_url, faction,
	account_level, account_xp, google_id, apple_id, runner_tier, created_at, updated_at, deleted_at`

func (r *PostgresRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	row := r.pool.QueryRow(ctx, fmt.Sprintf(`SELECT %s FROM users WHERE id = $1 AND deleted_at IS NULL`, userCols), id)
	return scanUser(row)
}

func (r *PostgresRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	row := r.pool.QueryRow(ctx, fmt.Sprintf(`SELECT %s FROM users WHERE email = $1 AND deleted_at IS NULL`, userCols), email)
	return scanUser(row)
}

func (r *PostgresRepo) GetByGoogleID(ctx context.Context, googleID string) (*domain.User, error) {
	row := r.pool.QueryRow(ctx, fmt.Sprintf(`SELECT %s FROM users WHERE google_id = $1 AND deleted_at IS NULL`, userCols), googleID)
	return scanUser(row)
}

func (r *PostgresRepo) GetByAppleID(ctx context.Context, appleID string) (*domain.User, error) {
	row := r.pool.QueryRow(ctx, fmt.Sprintf(`SELECT %s FROM users WHERE apple_id = $1 AND deleted_at IS NULL`, userCols), appleID)
	return scanUser(row)
}

func (r *PostgresRepo) GetByPhoneHash(ctx context.Context, hash string) (*domain.User, error) {
	row := r.pool.QueryRow(ctx, fmt.Sprintf(`SELECT %s FROM users WHERE phone_hash = $1 AND deleted_at IS NULL`, userCols), hash)
	return scanUser(row)
}

func (r *PostgresRepo) Create(ctx context.Context, user *domain.User) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO users (id, email, password_hash, display_name, phone_hash, avatar_url, faction,
			account_level, account_xp, google_id, apple_id, runner_tier, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		user.ID, user.Email, user.PasswordHash, user.DisplayName, user.PhoneHash,
		user.AvatarURL, user.Faction, user.AccountLevel, user.AccountXP,
		user.GoogleID, user.AppleID, user.RunnerTier, user.CreatedAt, user.UpdatedAt)
	return err
}

func (r *PostgresRepo) Update(ctx context.Context, user *domain.User) error {
	user.UpdatedAt = time.Now()
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET display_name=$2, phone_hash=$3, avatar_url=$4, faction=$5,
			runner_tier=$6, updated_at=$7 WHERE id=$1`,
		user.ID, user.DisplayName, user.PhoneHash, user.AvatarURL,
		user.Faction, user.RunnerTier, user.UpdatedAt)
	return err
}

func (r *PostgresRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	return err
}

func (r *PostgresRepo) Export(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return r.GetByID(ctx, id)
}

func (r *PostgresRepo) IncrementXP(ctx context.Context, id uuid.UUID, xp int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET account_xp = account_xp + $2 WHERE id = $1`, id, xp)
	return err
}

func (r *PostgresRepo) UpdateFaction(ctx context.Context, id uuid.UUID, faction domain.Faction) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET faction = $2, updated_at = NOW() WHERE id = $1`, id, faction)
	return err
}
