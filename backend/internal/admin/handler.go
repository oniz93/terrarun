package admin

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	"github.com/terrarun/backend/internal/middleware"
)

type Handler struct {
	db  *pgxpool.Pool
	rdb *redis.Client
}

func NewHandler(db *pgxpool.Pool, rdb *redis.Client) *Handler {
	return &Handler{db: db, rdb: rdb}
}

func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(h.adminOnly)

	r.Get("/stats", h.GetStats)
	r.Post("/users/{id}/ban", h.BanUser)
	r.Post("/system/clear-cache", h.ClearCache)
	r.Post("/system/run-migration", h.RunMigration)
	return r
}

func (h *Handler) adminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := middleware.UserIDFromContext(r.Context())
		var isAdmin bool
		err := h.db.QueryRow(r.Context(),
			`SELECT is_admin FROM users WHERE id = $1`, userID).Scan(&isAdmin)
		if err != nil || !isAdmin {
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	var stats struct {
		Users        int `json:"users"`
		TotalRuns    int `json:"total_runs"`
		TotalHexes   int `json:"total_hexes"`
		ActiveToday  int `json:"active_today"`
	}

	stats.Users = h.queryCount(r, `SELECT COUNT(*) FROM users`)
	stats.TotalRuns = h.queryCount(r, `SELECT COUNT(*) FROM runs`)
	stats.TotalHexes = h.queryCount(r, `SELECT COUNT(*) FROM hexes WHERE owned_by IS NOT NULL`)
	stats.ActiveToday = int(h.rdb.PFCount(r.Context(), "stats:active_today").Val())

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (h *Handler) queryCount(r *http.Request, query string) int {
	var count int
	h.db.QueryRow(r.Context(), query).Scan(&count)
	return count
}

func (h *Handler) BanUser(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid user id"}`, http.StatusBadRequest)
		return
	}

	_, err = h.db.Exec(r.Context(),
		`UPDATE users SET is_banned = true, updated_at = NOW() WHERE id = $1`, userID)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID.String()).Msg("ban user failed")
		http.Error(w, `{"error":"ban failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "banned"})
}

func (h *Handler) ClearCache(w http.ResponseWriter, r *http.Request) {
	err := h.rdb.FlushDB(r.Context()).Err()
	if err != nil {
		http.Error(w, `{"error":"clear cache failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "cache cleared"})
}

func (h *Handler) RunMigration(w http.ResponseWriter, r *http.Request) {
	log.Warn().Msg("admin triggered migration")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "migration triggered"})
}
