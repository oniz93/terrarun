package leaderboard

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/team", h.GetTeamLeaderboard)
	r.Get("/runners", h.GetRunnerLeaderboard)
	r.Get("/friends", h.GetFriendsLeaderboard)
	return r
}

func (h *Handler) GetTeamLeaderboard(w http.ResponseWriter, r *http.Request) {
	region := r.URL.Query().Get("region")
	if region == "" {
		region = "global"
	}
	entries, err := h.svc.GetTeamLeaderboard(r.Context(), region)
	if err != nil {
		http.Error(w, `{"error":"failed to get leaderboard"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"entries": entries})
}

func (h *Handler) GetRunnerLeaderboard(w http.ResponseWriter, r *http.Request) {
	season := r.URL.Query().Get("season")
	if season == "" {
		season = "current"
	}
	entries, err := h.svc.GetRunnerLeaderboard(r.Context(), season)
	if err != nil {
		http.Error(w, `{"error":"failed to get leaderboard"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"entries": entries})
}

func (h *Handler) GetFriendsLeaderboard(w http.ResponseWriter, r *http.Request) {
	entries, err := h.svc.GetFriendsLeaderboard(r.Context(), "")
	if err != nil {
		http.Error(w, `{"error":"failed to get leaderboard"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"entries": entries})
}
