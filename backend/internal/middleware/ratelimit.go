package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type rateLimitConfig struct {
	limit  int
	window time.Duration
}

func routeRateLimit(path string) *rateLimitConfig {
	switch {
	case strings.Contains(path, "/auth/register"):
		return &rateLimitConfig{limit: 5, window: time.Hour}
	case strings.Contains(path, "/auth/login"):
		return &rateLimitConfig{limit: 20, window: time.Hour}
	case strings.Contains(path, "/auth/refresh"):
		return &rateLimitConfig{limit: 10, window: time.Minute}
	case strings.Contains(path, "/runs/start"):
		return &rateLimitConfig{limit: 10, window: time.Hour}
	case strings.Contains(path, "/runs/end"):
		return &rateLimitConfig{limit: 10, window: time.Hour}
	case strings.Contains(path, "/territory"):
		return &rateLimitConfig{limit: 30, window: time.Minute}
	case strings.Contains(path, "/friends/add"):
		return &rateLimitConfig{limit: 20, window: time.Hour}
	}
	return nil
}

func CheckRateLimit(ctx context.Context, rdb *redis.Client, key string, limit int, window time.Duration) (bool, error) {
	now := time.Now().UnixNano()
	windowStart := now - window.Nanoseconds()

	pipe := rdb.Pipeline()
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprint(windowStart))
	pipe.ZCard(ctx, key)
	pipe.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: now})
	pipe.Expire(ctx, key, window)

	cmds, err := pipe.Exec(ctx)
	if err != nil {
		return false, err
	}
	count := cmds[1].(*redis.IntCmd).Val()
	return int(count) <= limit, nil
}

func RateLimit(rdb *redis.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cfg := routeRateLimit(r.URL.Path)
			if cfg == nil {
				next.ServeHTTP(w, r)
				return
			}

			ip := r.RemoteAddr
			key := fmt.Sprintf("ratelimit:%s:%s", ip, r.URL.Path)

			allowed, err := CheckRateLimit(r.Context(), rdb, key, cfg.limit, cfg.window)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			if !allowed {
				http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
