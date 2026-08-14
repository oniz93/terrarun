package auth

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/terrarun/backend/internal/domain"
)

type Repository interface {
	Create(ctx context.Context, sess *domain.Session) error
	GetByRefreshToken(ctx context.Context, token string) (*domain.Session, error)
	Revoke(ctx context.Context, id uuid.UUID) error
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
}

type PostgresRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

func (r *PostgresRepo) Create(ctx context.Context, sess *domain.Session) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO sessions (id, user_id, refresh_token, device_info, expires_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		sess.ID, sess.UserID, sess.RefreshToken, sess.DeviceInfo, sess.ExpiresAt)
	return err
}

func (r *PostgresRepo) GetByRefreshToken(ctx context.Context, token string) (*domain.Session, error) {
	var sess domain.Session
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, refresh_token, device_info, expires_at, created_at, revoked_at
		 FROM sessions WHERE refresh_token = $1`, token).
		Scan(&sess.ID, &sess.UserID, &sess.RefreshToken, &sess.DeviceInfo,
			&sess.ExpiresAt, &sess.CreatedAt, &sess.RevokedAt)
	if err != nil {
		return nil, err
	}
	return &sess, nil
}

func (r *PostgresRepo) Revoke(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE sessions SET revoked_at = NOW() WHERE id = $1`, id)
	return err
}

func (r *PostgresRepo) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE sessions SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`, userID)
	return err
}
