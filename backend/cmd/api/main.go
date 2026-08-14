package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/terrarun/backend/internal/admin"
	"github.com/terrarun/backend/internal/auth"
	"github.com/terrarun/backend/internal/bot"
	"github.com/terrarun/backend/internal/config"
	"github.com/terrarun/backend/internal/friend"
	"github.com/terrarun/backend/internal/leaderboard"
	"github.com/terrarun/backend/internal/middleware"
	"github.com/terrarun/backend/internal/notification"
	"github.com/terrarun/backend/internal/points"
	"github.com/terrarun/backend/internal/run"
	"github.com/terrarun/backend/internal/season"
	"github.com/terrarun/backend/internal/storage"
	"github.com/terrarun/backend/internal/territory"
	"github.com/terrarun/backend/internal/user"
	"github.com/terrarun/backend/internal/websocket"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Info().Msg("terrarun api starting")

	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	db := mustConnectDB(cfg)
	defer db.Close()

	rdb := mustConnectRedis(cfg)
	defer rdb.Close()

	storageClient, err := storage.NewRustFS(cfg.StorageEndpoint, cfg.StorageAccessKey, cfg.StorageSecretKey, cfg.StorageBucket, cfg.StorageUseSSL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to init storage")
	}

	jwtSvc := auth.NewJWTService(cfg.JWTSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	authSessionRepo := auth.NewPostgresRepo(db)
	userRepo := user.NewPostgresRepo(db)
	authSvc := auth.NewService(userRepo, authSessionRepo, jwtSvc, cfg)
	authH := auth.NewHandler(authSvc)

	userSvc := user.NewService(userRepo, rdb, storageClient)
	userH := user.NewHandler(userSvc)

	runRepo := run.NewPostgresRepo(db)
	pointsCalc := points.NewCalculator(cfg)
	cheatDetector := run.NewCheatDetector(cfg)
	statsCalc := run.NewStatsCalculator()
	geoRepo := territory.NewPostgresRepo(db)
	territorySvc := territory.NewService(geoRepo, rdb, cfg)

	wsHub := websocket.NewHub()
	go wsHub.Run()

	runSvc := run.NewService(runRepo, territorySvc, cheatDetector, statsCalc, pointsCalc, wsHub, userRepo, cfg)
	runH := run.NewHandler(runSvc)

	territoryH := territory.NewHandler(territorySvc)

	friendRepo := friend.NewPostgresRepo(db)
	friendSvc := friend.NewService(friendRepo, userRepo, rdb)
	friendH := friend.NewHandler(friendSvc)

	leaderboardRepo := leaderboard.NewRedisRepo(rdb)
	leaderboardSvc := leaderboard.NewService(leaderboardRepo)
	leaderboardH := leaderboard.NewHandler(leaderboardSvc)

	botRepo := bot.NewPostgresRepo(db)
	botSvc := bot.NewService(botRepo, db, rdb, geoRepo)
	botSched := bot.NewScheduler(botSvc, 15*time.Minute)
	go botSched.Start(context.Background())

	decaySched := territory.NewDecayScheduler(territorySvc, 10000, time.Hour)
	go decaySched.Start(context.Background())

	notifSvc := notification.NewService(db, rdb, wsHub)

	seasonRepo := season.NewPostgresRepo(db)
	seasonSvc := season.NewService(seasonRepo)
	seasonH := season.NewHandler(seasonSvc)

	adminH := admin.NewHandler(db, rdb)

	r := chi.NewRouter()

	r.Use(chimw.RealIP)
	r.Use(middleware.RequestID)
	r.Use(middleware.Recovery)
	r.Use(middleware.Logger)
	r.Use(middleware.CORS)
	r.Use(chimw.Timeout(30 * time.Second))

	r.Get("/health", healthHandler(db, rdb))
	r.Get("/health/ready", healthHandler(db, rdb))

	if cfg.MapboxToken != "" {
		r.Route("/mapbox", func(r chi.Router) {
			r.Use(middleware.RateLimit(rdb))
			r.Get("/{z}/{x}/{y}.mvt", mapboxTileProxy(cfg))
			r.Get("/{z}/{x}/{y}.png", mapboxTileProxy(cfg))
			r.Get("/{z}/{x}/{y}", mapboxTileProxy(cfg))
		})
	}

	r.Route("/auth", func(r chi.Router) {
		r.Use(middleware.RateLimit(rdb))
		r.Mount("/", authH.Routes())
	})

	r.Route("/ws", func(r chi.Router) {
		r.Get("/", websocket.WsHandler(wsHub, cfg.JWTSecret))
	})

	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(cfg.JWTSecret))
		r.Use(middleware.RateLimit(rdb))

		r.Route("/users", func(r chi.Router) { r.Mount("/", userH.Routes()) })
		r.Route("/runs", func(r chi.Router) { r.Mount("/", runH.Routes()) })
		r.Route("/territory", func(r chi.Router) { r.Mount("/", territoryH.Routes()) })
		r.Route("/friends", func(r chi.Router) { r.Mount("/", friendH.Routes()) })
		r.Route("/leaderboard", func(r chi.Router) { r.Mount("/", leaderboardH.Routes()) })
		r.Route("/seasons", func(r chi.Router) { r.Mount("/", seasonH.Routes()) })
		r.Route("/admin", func(r chi.Router) { r.Mount("/", adminH.Routes()) })

		r.Route("/notifications", func(r chi.Router) {
			r.Get("/", notifGetUnread(notifSvc))
			r.Post("/{id}/read", notifMarkRead(notifSvc))
		})
	})

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	go func() {
		log.Info().Str("port", cfg.Port).Msg("listening")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server error")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	srv.Shutdown(ctx)
	log.Info().Msg("server stopped")
}

func mapboxTileProxy(cfg *config.Config) http.HandlerFunc {
	target, _ := url.Parse("https://api.mapbox.com")
	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director
	proxy.Director = func(r *http.Request) {
		originalDirector(r)
		r.Host = "api.mapbox.com"
		r.URL.Scheme = "https"
		r.URL.Host = "api.mapbox.com"
		q := r.URL.Query()
		q.Set("access_token", cfg.MapboxToken)
		r.URL.RawQuery = q.Encode()
	}
	return proxy.ServeHTTP
}

func notifGetUnread(svc *notification.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := middleware.UserIDFromContext(r.Context())
		notifs, err := svc.GetUnread(r.Context(), userID)
		if err != nil {
			http.Error(w, `{"error":"get notifications failed"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"notifications": notifs})
	}
}

func notifMarkRead(svc *notification.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := middleware.UserIDFromContext(r.Context())
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			http.Error(w, `{"error":"invalid notification id"}`, http.StatusBadRequest)
			return
		}
		if err := svc.MarkRead(r.Context(), userID, id); err != nil {
			http.Error(w, `{"error":"mark read failed"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "read"})
	}
}

func mustConnectDB(cfg *config.Config) *pgxpool.Pool {
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("parse db config")
	}
	poolCfg.MaxConns = int32(cfg.MaxDBConns)
	poolCfg.MinConns = int32(cfg.MinDBConns)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		log.Fatal().Err(err).Msg("connect db")
	}
	if err := pool.Ping(ctx); err != nil {
		log.Fatal().Err(err).Msg("ping db")
	}
	log.Info().Msg("database connected")
	return pool
}

func mustConnectRedis(cfg *config.Config) *redis.Client {
	opts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatal().Err(err).Msg("parse redis url")
	}
	opts.MaxRetries = cfg.RedisMaxRetries
	rdb := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatal().Err(err).Msg("ping redis")
	}
	log.Info().Msg("redis connected")
	return rdb
}

func healthHandler(db *pgxpool.Pool, rdb *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbOK := true
		if err := db.Ping(r.Context()); err != nil {
			dbOK = false
		}
		redisOK := true
		if err := rdb.Ping(r.Context()).Err(); err != nil {
			redisOK = false
		}
		if !dbOK || !redisOK {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, `{"status":"degraded","db":%t,"redis":%t}`, dbOK, redisOK)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ok","db":true,"redis":true}`)
	}
}
