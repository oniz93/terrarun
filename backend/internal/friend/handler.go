package friend

import (
	"encoding/json"
	"errors"
	"net/http"

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

const (
	statusCreated  = http.StatusCreated
	statusOK       = http.StatusOK
	statusNotFound = http.StatusNotFound
)

func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Post("/add", h.AddFriend)
	r.Post("/{id}/accept", h.AcceptFriend)
	r.Delete("/{id}", h.RejectFriend)
	r.Get("/", h.GetFriends)
	r.Get("/pending", h.GetPending)
	r.Get("/feed", h.GetFeed)
	return r
}

func (h *Handler) AddFriend(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	var req struct {
		UserID uuid.UUID `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}
	if err := h.svc.AddFriend(r.Context(), userID, req.UserID); err != nil {
		switch {
		case errors.Is(err, ErrAlreadyFriends):
			http.Error(w, `{"error":"already friends"}`, http.StatusConflict)
		case errors.Is(err, ErrSelfRequest):
			http.Error(w, `{"error":"cannot friend yourself"}`, http.StatusBadRequest)
		default:
			http.Error(w, `{"error":"add friend failed"}`, http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "pending"})
}

func (h *Handler) AcceptFriend(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	friendID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid friend id"}`, http.StatusBadRequest)
		return
	}
	if err := h.svc.AcceptFriend(r.Context(), userID, friendID); err != nil {
		http.Error(w, `{"error":"no pending request"}`, statusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
}

func (h *Handler) RejectFriend(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	friendID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid friend id"}`, http.StatusBadRequest)
		return
	}
	if err := h.svc.RejectFriend(r.Context(), userID, friendID); err != nil {
		http.Error(w, `{"error":"reject failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "rejected"})
}

func (h *Handler) GetFriends(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	friends, err := h.svc.GetFriends(r.Context(), userID)
	if err != nil {
		http.Error(w, `{"error":"get friends failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"friends": friends})
}

func (h *Handler) GetPending(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	pending, err := h.svc.GetPending(r.Context(), userID)
	if err != nil {
		http.Error(w, `{"error":"get pending failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"pending": pending})
}

func (h *Handler) GetFeed(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	feed, err := h.svc.GetFeed(r.Context(), userID)
	if err != nil {
		http.Error(w, `{"error":"get feed failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"feed": feed})
}
