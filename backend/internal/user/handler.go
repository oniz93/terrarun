package user

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

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
	r.Get("/me", h.GetMe)
	r.Patch("/me", h.UpdateMe)
	r.Post("/me/avatar", h.UploadAvatar)
	r.Post("/me/export", h.ExportData)
	r.Post("/me/delete", h.DeleteAccount)
	r.Post("/contacts/match", h.MatchContacts)
	r.Get("/{id}", h.GetPublic)
	return r
}

func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	user, err := h.svc.GetMe(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusNotFound, "user not found")
		return
	}
	respondJSON(w, http.StatusOK, user)
}

func (h *Handler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request")
		return
	}
	user, err := h.svc.UpdateMe(r.Context(), userID, req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "update failed")
		return
	}
	respondJSON(w, http.StatusOK, user)
}

func (h *Handler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	r.ParseMultipartForm(10 << 20)
	file, header, err := r.FormFile("avatar")
	if err != nil {
		respondError(w, http.StatusBadRequest, "avatar file required")
		return
	}
	defer file.Close()

	url, err := h.svc.UploadAvatar(r.Context(), userID, file, header)
	if err != nil {
		log.Error().Err(err).Msg("avatar upload failed")
		respondError(w, http.StatusInternalServerError, "upload failed")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"avatar_url": url})
}

func (h *Handler) GetPublic(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	userID, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	user, err := h.svc.GetPublic(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusNotFound, "user not found")
		return
	}
	respondJSON(w, http.StatusOK, user)
}

func (h *Handler) MatchContacts(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Hashes []string `json:"hashes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request")
		return
	}
	result, err := h.svc.MatchContacts(r.Context(), req.Hashes)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "contacts match failed")
		return
	}
	respondJSON(w, http.StatusOK, result)
}

func (h *Handler) ExportData(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	data, err := h.svc.ExportData(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusNotFound, "user not found")
		return
	}
	respondJSON(w, http.StatusOK, data)
}

func (h *Handler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if err := h.svc.DeleteAccount(r.Context(), userID); err != nil {
		respondError(w, http.StatusInternalServerError, "delete failed")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
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
