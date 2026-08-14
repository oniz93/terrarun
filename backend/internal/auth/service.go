package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"

	"github.com/terrarun/backend/internal/config"
	"github.com/terrarun/backend/internal/domain"
	"github.com/terrarun/backend/internal/user"
)

var (
	ErrEmailTaken         = errors.New("email already taken")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrOAuthOnlyAccount   = errors.New("account uses OAuth, sign in with Google or Apple")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenReused        = errors.New("refresh token reused")
)

type Service struct {
	userRepo user.Repository
	sessRepo Repository
	jwt      *JWTService
	cfg      *config.Config
}

func NewService(userRepo user.Repository, sessRepo Repository, jwt *JWTService, cfg *config.Config) *Service {
	return &Service{
		userRepo: userRepo,
		sessRepo: sessRepo,
		jwt:      jwt,
		cfg:      cfg,
	}
}

func (s *Service) Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error) {
	existing, _ := s.userRepo.GetByEmail(ctx, req.Email)
	if existing != nil {
		return nil, ErrEmailTaken
	}

	var pwHash *string
	if req.Password != nil {
		h, err := bcrypt.GenerateFromPassword([]byte(*req.Password), s.cfg.BcryptCost)
		if err != nil {
			return nil, fmt.Errorf("hash password: %w", err)
		}
		s := string(h)
		pwHash = &s
	}

	user := domain.User{
		ID:           uuid.New(),
		Email:        req.Email,
		PasswordHash: pwHash,
		DisplayName:  req.DisplayName,
		PhoneHash:    req.PhoneHash,
		Faction:      req.Faction,
		RunnerTier:   domain.TierFree,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := s.userRepo.Create(ctx, &user); err != nil {
		return nil, err
	}
	log.Info().Str("user_id", user.ID.String()).Msg("user registered")
	return s.createAuthResponse(ctx, user)
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (*AuthResponse, error) {
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	if user.PasswordHash == nil {
		return nil, ErrOAuthOnlyAccount
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}
	return s.createAuthResponse(ctx, *user)
}

func (s *Service) GoogleLogin(ctx context.Context, req GoogleLoginRequest) (*AuthResponse, error) {
	payload, err := verifyGoogleIDToken(ctx, req.IDToken, s.cfg.GoogleClientID)
	if err != nil {
		return nil, ErrInvalidToken
	}
	user, err := s.userRepo.GetByGoogleID(ctx, payload.Subject)
	if err != nil {
		user = &domain.User{
			ID:          uuid.New(),
			Email:       payload.Email,
			DisplayName: payload.Name,
			GoogleID:    &payload.Subject,
			Faction:     req.Faction,
			RunnerTier:  domain.TierFree,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		if err := s.userRepo.Create(ctx, user); err != nil {
			return nil, err
		}
		log.Info().Str("user_id", user.ID.String()).Msg("user registered via google")
	}
	return s.createAuthResponse(ctx, *user)
}

func (s *Service) AppleLogin(ctx context.Context, req AppleLoginRequest) (*AuthResponse, error) {
	payload, err := verifyAppleIdentityToken(ctx, req.IdentityToken, s.cfg.AppleClientID)
	if err != nil {
		return nil, ErrInvalidToken
	}
	uid := payload.Subject
	if req.UserIdentifier != "" {
		uid = req.UserIdentifier
	}
	user, err := s.userRepo.GetByAppleID(ctx, uid)
	if err != nil {
		user = &domain.User{
			ID:          uuid.New(),
			Email:       payload.Email,
			DisplayName: req.DisplayName,
			AppleID:     &uid,
			Faction:     req.Faction,
			RunnerTier:  domain.TierFree,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		if err := s.userRepo.Create(ctx, user); err != nil {
			return nil, err
		}
		log.Info().Str("user_id", user.ID.String()).Msg("user registered via apple")
	}
	return s.createAuthResponse(ctx, *user)
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (*AuthResponse, error) {
	sess, err := s.sessRepo.GetByRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, ErrInvalidToken
	}
	if sess.RevokedAt != nil {
		s.sessRepo.RevokeAllForUser(ctx, sess.UserID)
		log.Warn().Str("user_id", sess.UserID.String()).Msg("refresh token reuse detected")
		return nil, ErrTokenReused
	}
	s.sessRepo.Revoke(ctx, sess.ID)

	user, err := s.userRepo.GetByID(ctx, sess.UserID)
	if err != nil {
		return nil, ErrInvalidToken
	}
	log.Info().Str("user_id", user.ID.String()).Msg("token refreshed")
	return s.createAuthResponse(ctx, *user)
}

func (s *Service) createAuthResponse(ctx context.Context, user domain.User) (*AuthResponse, error) {
	accessToken, err := s.jwt.GenerateAccessToken(user)
	if err != nil {
		return nil, err
	}

	refreshToken, expiresAt := s.jwt.GenerateRefreshToken()
	sess := domain.Session{
		ID:           uuid.New(),
		UserID:       user.ID,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
	}
	if err := s.sessRepo.Create(ctx, &sess); err != nil {
		return nil, err
	}

	return &AuthResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
