package auth

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/terrarun/backend/internal/domain"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Post("/register", h.Register)
	r.Post("/login", h.Login)
	r.Post("/google", h.GoogleLogin)
	r.Post("/apple", h.AppleLogin)
	r.Post("/refresh", h.Refresh)
	return r
}

type RegisterRequest struct {
	Email       string          `json:"email"`
	Password    *string         `json:"password,omitempty"`
	DisplayName string          `json:"display_name"`
	PhoneHash   *string         `json:"phone_hash,omitempty"`
	Faction     *domain.Faction `json:"faction,omitempty"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type GoogleLoginRequest struct {
	IDToken   string          `json:"id_token"`
	Faction   *domain.Faction `json:"faction,omitempty"`
	PhoneHash *string         `json:"phone_hash,omitempty"`
}

type AppleLoginRequest struct {
	IdentityToken  string          `json:"identity_token"`
	UserIdentifier string          `json:"user_identifier"`
	DisplayName    string          `json:"display_name"`
	Faction        *domain.Faction `json:"faction,omitempty"`
}

type AuthResponse struct {
	User         domain.User `json:"user"`
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Email == "" || req.DisplayName == "" {
		respondError(w, http.StatusBadRequest, "email and display_name required")
		return
	}
	resp, err := h.svc.Register(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrEmailTaken) {
			respondError(w, http.StatusConflict, "email already registered")
			return
		}
		respondError(w, http.StatusInternalServerError, "registration failed")
		return
	}
	respondJSON(w, http.StatusCreated, resp)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	resp, err := h.svc.Login(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) || errors.Is(err, ErrOAuthOnlyAccount) {
			respondError(w, http.StatusUnauthorized, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, "login failed")
		return
	}
	respondJSON(w, http.StatusOK, resp)
}

func (h *Handler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	var req GoogleLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	resp, err := h.svc.GoogleLogin(r.Context(), req)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "google login failed")
		return
	}
	respondJSON(w, http.StatusOK, resp)
}

func (h *Handler) AppleLogin(w http.ResponseWriter, r *http.Request) {
	var req AppleLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	resp, err := h.svc.AppleLogin(r.Context(), req)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "apple login failed")
		return
	}
	respondJSON(w, http.StatusOK, resp)
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	resp, err := h.svc.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, ErrTokenReused) {
			respondError(w, http.StatusUnauthorized, "token reused, all sessions revoked")
			return
		}
		respondError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}
	respondJSON(w, http.StatusOK, resp)
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
