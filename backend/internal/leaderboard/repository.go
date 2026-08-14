package leaderboard

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type RedisRepo struct {
	rdb *redis.Client
}

func NewRedisRepo(rdb *redis.Client) *RedisRepo {
	return &RedisRepo{rdb: rdb}
}

const (
	teamLeaderboardKey    = "leaderboard:team:%s"
	runnerLeaderboardKey  = "leaderboard:runner:%s:%s"
	friendsLeaderboardKey = "leaderboard:friends:%s"
)

func (r *RedisRepo) UpdateTeamScore(ctx context.Context, region, faction string, delta int64) error {
	return r.rdb.ZIncrBy(ctx, fmt.Sprintf(teamLeaderboardKey, region), float64(delta), faction).Err()
}

func (r *RedisRepo) UpdateRunnerScore(ctx context.Context, seasonID, userID string, points int64) error {
	return r.rdb.ZIncrBy(ctx, fmt.Sprintf(runnerLeaderboardKey, seasonID, "global"), float64(points), userID).Err()
}

func (r *RedisRepo) GetTeamLeaderboard(ctx context.Context, region string, topN int) ([]redis.Z, error) {
	return r.rdb.ZRevRangeWithScores(ctx, fmt.Sprintf(teamLeaderboardKey, region), 0, int64(topN-1)).Result()
}

func (r *RedisRepo) GetRunnerLeaderboard(ctx context.Context, seasonID string, topN int) ([]redis.Z, error) {
	return r.rdb.ZRevRangeWithScores(ctx, fmt.Sprintf(runnerLeaderboardKey, seasonID, "global"), 0, int64(topN-1)).Result()
}

func (r *RedisRepo) GetUserRank(ctx context.Context, seasonID, userID string) (int64, error) {
	rank, err := r.rdb.ZRevRank(ctx, fmt.Sprintf(runnerLeaderboardKey, seasonID, "global"), userID).Result()
	if err != nil {
		return 0, err
	}
	return rank + 1, nil
}

func (r *RedisRepo) UpdateFriendsLeaderboard(ctx context.Context, userID uuid.UUID, friendIDs []uuid.UUID, score int64) error {
	key := fmt.Sprintf(friendsLeaderboardKey, userID.String())
	pipe := r.rdb.Pipeline()
	for _, fid := range friendIDs {
		pipe.ZIncrBy(ctx, key, float64(score), fid.String())
	}
	pipe.Expire(ctx, key, 24*time.Hour)
	_, err := pipe.Exec(ctx)
	return err
}

func (r *RedisRepo) GetFriendsLeaderboard(ctx context.Context, userID uuid.UUID, topN int) ([]redis.Z, error) {
	return r.rdb.ZRevRangeWithScores(ctx, fmt.Sprintf(friendsLeaderboardKey, userID.String()), 0, int64(topN-1)).Result()
}

func (r *RedisRepo) SetUserActive(ctx context.Context, userID uuid.UUID) error {
	key := fmt.Sprintf("presence:user:%s", userID.String())
	return r.rdb.Set(ctx, key, time.Now().Unix(), 5*time.Minute).Err()
}
