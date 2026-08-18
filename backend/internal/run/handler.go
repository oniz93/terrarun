package run

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

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
	r.Post("/start", h.StartRun)
	r.Post("/{id}/end", h.EndRun)
	r.Get("/", h.ListRuns)
	r.Get("/{id}", h.GetRun)
	r.Get("/{id}/gps", h.GetGPSPoints)
	return r
}

func (h *Handler) StartRun(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	var req StartRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	run, err := h.svc.StartRun(r.Context(), userID, req)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, run)
}

func (h *Handler) EndRun(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	runID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid run id")
		return
	}
	var req EndRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	summary, err := h.svc.EndRun(r.Context(), userID, runID, req)
	if err != nil {
		if errors.Is(err, ErrNotOwner) {
			respondError(w, http.StatusForbidden, "not your run")
			return
		}
		if errors.Is(err, ErrRunNotActive) {
			respondError(w, http.StatusConflict, "run already ended")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to end run")
		return
	}
	respondJSON(w, http.StatusOK, summary)
}

func (h *Handler) GetRun(w http.ResponseWriter, r *http.Request) {
	runID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid run id")
		return
	}
	userID := middleware.UserIDFromContext(r.Context())
	run, err := h.svc.GetRun(r.Context(), userID, runID)
	if err != nil {
		if errors.Is(err, ErrNotOwner) {
			respondError(w, http.StatusForbidden, "not your run")
			return
		}
		respondError(w, http.StatusNotFound, "run not found")
		return
	}
	respondJSON(w, http.StatusOK, run)
}

func (h *Handler) ListRuns(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	runs, total, err := h.svc.ListRuns(r.Context(), userID, page, perPage)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list runs")
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"runs":     runs,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}

func (h *Handler) GetGPSPoints(w http.ResponseWriter, r *http.Request) {
	runID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid run id")
		return
	}
	userID := middleware.UserIDFromContext(r.Context())
	points, err := h.svc.GetGPSPoints(r.Context(), userID, runID)
	if err != nil {
		if errors.Is(err, ErrNotOwner) {
			respondError(w, http.StatusForbidden, "not your run")
			return
		}
		respondError(w, http.StatusNotFound, "gps points not found")
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"gps_points": points,
	})
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
