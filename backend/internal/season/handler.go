package season

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/terrarun/backend/internal/domain"
	"github.com/terrarun/backend/internal/middleware"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/current", h.GetCurrent)
	r.Get("/", h.List)
	r.Get("/{id}", h.GetByID)
	r.Post("/join", h.Join)
	r.Get("/{id}/leaderboard", h.GetLeaderboard)
	return r
}

func (h *Handler) GetCurrent(w http.ResponseWriter, r *http.Request) {
	season, err := h.svc.GetCurrentSeason(r.Context())
	if err != nil {
		http.Error(w, `{"error":"get current season failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(season)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	seasons, err := h.svc.ListSeasons(r.Context())
	if err != nil {
		http.Error(w, `{"error":"list seasons failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"seasons": seasons})
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid season id"}`, http.StatusBadRequest)
		return
	}

	season, err := h.svc.GetSeasonByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"season not found"}`, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(season)
}

func (h *Handler) Join(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	var req struct {
		Faction string `json:"faction"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}

	if err := h.svc.JoinSeason(r.Context(), userID, domain.Faction(req.Faction)); err != nil {
		http.Error(w, `{"error":"join season failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "joined"})
}

func (h *Handler) GetLeaderboard(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid season id"}`, http.StatusBadRequest)
		return
	}

	participants, err := h.svc.GetLeaderboard(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"get leaderboard failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"participants": participants})
}
