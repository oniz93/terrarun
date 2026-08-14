package friend

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/terrarun/backend/internal/domain"
)

type Repository interface {
	AddRequest(ctx context.Context, userID, friendID uuid.UUID) error
	Accept(ctx context.Context, userID, friendID uuid.UUID) error
	Reject(ctx context.Context, userID, friendID uuid.UUID) error
	Block(ctx context.Context, userID, friendID uuid.UUID) error
	GetFriends(ctx context.Context, userID uuid.UUID) ([]domain.Friend, error)
	GetPending(ctx context.Context, userID uuid.UUID) ([]domain.Friend, error)
	GetByUserPair(ctx context.Context, userID, friendID uuid.UUID) (*domain.Friend, error)
	GetFriendIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
}

type PostgresRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

func (r *PostgresRepo) AddRequest(ctx context.Context, userID, friendID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO friends (user_id, friend_id, status) VALUES ($1, $2, 'pending')
		 ON CONFLICT (user_id, friend_id) DO UPDATE SET status = 'pending', updated_at = NOW()`,
		userID, friendID)
	return err
}

func (r *PostgresRepo) Accept(ctx context.Context, userID, friendID uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`UPDATE friends SET status = 'accepted', updated_at = NOW()
		 WHERE user_id = $1 AND friend_id = $2`, userID, friendID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO friends (user_id, friend_id, status)
		 VALUES ($1, $2, 'accepted')
		 ON CONFLICT (user_id, friend_id) DO UPDATE SET status = 'accepted', updated_at = NOW()`,
		friendID, userID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *PostgresRepo) Reject(ctx context.Context, userID, friendID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM friends WHERE user_id = $1 AND friend_id = $2`,
		userID, friendID)
	return err
}

func (r *PostgresRepo) Block(ctx context.Context, userID, friendID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE friends SET status = 'blocked', updated_at = NOW()
		 WHERE user_id = $1 AND friend_id = $2`,
		userID, friendID)
	return err
}

func scanFriend(row pgx.Row) (*domain.Friend, error) {
	var f domain.Friend
	err := row.Scan(&f.UserID, &f.FriendID, &f.Status, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *PostgresRepo) GetByUserPair(ctx context.Context, userID, friendID uuid.UUID) (*domain.Friend, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT user_id, friend_id, status, created_at, updated_at
		 FROM friends WHERE user_id = $1 AND friend_id = $2`,
		userID, friendID)
	return scanFriend(row)
}

func (r *PostgresRepo) GetFriends(ctx context.Context, userID uuid.UUID) ([]domain.Friend, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT f.user_id, f.friend_id, f.status, f.created_at, f.updated_at
		 FROM friends f
		 JOIN friends g ON g.user_id = f.friend_id AND g.friend_id = f.user_id AND g.status = 'accepted'
		 WHERE f.user_id = $1 AND f.status = 'accepted'`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var friends []domain.Friend
	for rows.Next() {
		var f domain.Friend
		if err := rows.Scan(&f.UserID, &f.FriendID, &f.Status, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, err
		}
		friends = append(friends, f)
	}
	return friends, nil
}

func (r *PostgresRepo) GetPending(ctx context.Context, userID uuid.UUID) ([]domain.Friend, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT user_id, friend_id, status, created_at, updated_at
		 FROM friends WHERE friend_id = $1 AND status = 'pending'`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var friends []domain.Friend
	for rows.Next() {
		var f domain.Friend
		if err := rows.Scan(&f.UserID, &f.FriendID, &f.Status, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, err
		}
		friends = append(friends, f)
	}
	return friends, nil
}

func (r *PostgresRepo) GetFriendIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT friend_id FROM friends WHERE user_id = $1 AND status = 'accepted'`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}


