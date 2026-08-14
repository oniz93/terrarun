package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/terrarun/backend/internal/config"
	"github.com/terrarun/backend/internal/domain"
	"github.com/terrarun/backend/internal/user"
	"golang.org/x/crypto/bcrypt"
)

type mockUserRepo struct {
	user.Repository
	users map[string]*domain.User

	getByEmailFn func(ctx context.Context, email string) (*domain.User, error)
	createFn     func(ctx context.Context, user *domain.User) error
	getByIDFn    func(ctx context.Context, id uuid.UUID) (*domain.User, error)
}

func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if m.getByEmailFn != nil {
		return m.getByEmailFn(ctx, email)
	}
	u, ok := m.users[email]
	if !ok {
		return nil, errors.New("not found")
	}
	return u, nil
}

func (m *mockUserRepo) Create(ctx context.Context, user *domain.User) error {
	if m.createFn != nil {
		return m.createFn(ctx, user)
	}
	if m.users == nil {
		m.users = make(map[string]*domain.User)
	}
	m.users[user.Email] = user
	return nil
}

func (m *mockUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, errors.New("not found")
}

func (m *mockUserRepo) GetByGoogleID(ctx context.Context, googleID string) (*domain.User, error) {
	for _, u := range m.users {
		if u.GoogleID != nil && *u.GoogleID == googleID {
			return u, nil
		}
	}
	return nil, errors.New("not found")
}

func (m *mockUserRepo) GetByAppleID(ctx context.Context, appleID string) (*domain.User, error) {
	for _, u := range m.users {
		if u.AppleID != nil && *u.AppleID == appleID {
			return u, nil
		}
	}
	return nil, errors.New("not found")
}

type mockSessRepo struct {
	Repository
	sessions []*domain.Session

	getByRefreshTokenFn func(ctx context.Context, token string) (*domain.Session, error)
	createFn           func(ctx context.Context, sess *domain.Session) error
	revokeFn           func(ctx context.Context, id uuid.UUID) error
	revokeAllFn        func(ctx context.Context, userID uuid.UUID) error
}

func (m *mockSessRepo) Create(ctx context.Context, sess *domain.Session) error {
	if m.createFn != nil {
		return m.createFn(ctx, sess)
	}
	m.sessions = append(m.sessions, sess)
	return nil
}

func (m *mockSessRepo) GetByRefreshToken(ctx context.Context, token string) (*domain.Session, error) {
	if m.getByRefreshTokenFn != nil {
		return m.getByRefreshTokenFn(ctx, token)
	}
	for _, s := range m.sessions {
		if s.RefreshToken == token {
			return s, nil
		}
	}
	return nil, errors.New("not found")
}

func (m *mockSessRepo) Revoke(ctx context.Context, id uuid.UUID) error {
	if m.revokeFn != nil {
		return m.revokeFn(ctx, id)
	}
	for _, s := range m.sessions {
		if s.ID == id {
			now := time.Now()
			s.RevokedAt = &now
			return nil
		}
	}
	return nil
}

func (m *mockSessRepo) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	if m.revokeAllFn != nil {
		return m.revokeAllFn(ctx, userID)
	}
	now := time.Now()
	for _, s := range m.sessions {
		if s.UserID == userID && s.RevokedAt == nil {
			s.RevokedAt = &now
		}
	}
	return nil
}

func newTestAuthService(t *testing.T, overrides ...func(*config.Config)) (*Service, *mockUserRepo, *mockSessRepo) {
	t.Helper()
	cfg := &config.Config{
		BcryptCost:       bcrypt.MinCost,
		JWTSecret:        "test-secret-for-auth-svc",
		AccessTokenTTL:   15 * time.Minute,
		RefreshTokenTTL:  7 * 24 * time.Hour,
	}
	for _, fn := range overrides {
		fn(cfg)
	}
	userRepo := &mockUserRepo{users: make(map[string]*domain.User)}
	sessRepo := &mockSessRepo{}
	jwtSvc := NewJWTService(cfg.JWTSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	svc := NewService(userRepo, sessRepo, jwtSvc, cfg)
	return svc, userRepo, sessRepo
}

func TestRegister_Success(t *testing.T) {
	svc, _, _ := newTestAuthService(t)
	faction := domain.FactionNeon
	pw := "securepass123"
	resp, err := svc.Register(context.Background(), RegisterRequest{
		Email:       "new@test.com",
		Password:    &pw,
		DisplayName: "NewUser",
		Faction:     &faction,
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if resp.User.Email != "new@test.com" {
		t.Errorf("email = %s, want new@test.com", resp.User.Email)
	}
	if resp.User.Faction == nil || *resp.User.Faction != domain.FactionNeon {
		t.Error("expected neon faction")
	}
	if resp.User.RunnerTier != domain.TierFree {
		t.Errorf("tier = %s, want free", resp.User.RunnerTier)
	}
	if resp.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
	if resp.RefreshToken == "" {
		t.Error("expected non-empty refresh token")
	}
	if resp.User.PasswordHash == nil || *resp.User.PasswordHash == "" {
		t.Error("expected password hash")
	}
}

func TestRegister_EmailTaken(t *testing.T) {
	svc, ur, _ := newTestAuthService(t)
	pw := "pass123"
	ur.users["taken@test.com"] = &domain.User{Email: "taken@test.com"}

	_, err := svc.Register(context.Background(), RegisterRequest{
		Email:       "taken@test.com",
		Password:    &pw,
		DisplayName: "Dup",
	})
	if err != ErrEmailTaken {
		t.Fatalf("expected ErrEmailTaken, got %v", err)
	}
}

func TestRegister_OAuthWithoutPassword(t *testing.T) {
	svc, _, _ := newTestAuthService(t)
	faction := domain.FactionNeon
	resp, err := svc.Register(context.Background(), RegisterRequest{
		Email:       "oauth@test.com",
		DisplayName: "OAuthUser",
		Faction:     &faction,
	})
	if err != nil {
		t.Fatalf("Register (OAuth) failed: %v", err)
	}
	if resp.User.PasswordHash != nil {
		t.Error("expected nil password hash for OAuth user")
	}
}

func TestLogin_Success(t *testing.T) {
	svc, ur, _ := newTestAuthService(t)
	pw := "correctpw"
	hash, _ := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.MinCost)
	h := string(hash)
	ur.users["login@test.com"] = &domain.User{
		ID:           uuid.New(),
		Email:        "login@test.com",
		PasswordHash: &h,
		DisplayName:  "LoginUser",
		RunnerTier:   domain.TierFree,
	}

	resp, err := svc.Login(context.Background(), LoginRequest{
		Email:    "login@test.com",
		Password: "correctpw",
	})
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if resp.User.Email != "login@test.com" {
		t.Errorf("email = %s, want login@test.com", resp.User.Email)
	}
	if resp.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	svc, ur, _ := newTestAuthService(t)
	pw := "realpw"
	hash, _ := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.MinCost)
	h := string(hash)
	ur.users["wrong@test.com"] = &domain.User{
		Email:        "wrong@test.com",
		PasswordHash: &h,
	}

	_, err := svc.Login(context.Background(), LoginRequest{
		Email:    "wrong@test.com",
		Password: "wrongpw",
	})
	if err != ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	svc, _, _ := newTestAuthService(t)
	_, err := svc.Login(context.Background(), LoginRequest{
		Email:    "nobody@test.com",
		Password: "any",
	})
	if err != ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_OAuthOnly(t *testing.T) {
	svc, ur, _ := newTestAuthService(t)
	ur.users["oauth@test.com"] = &domain.User{
		Email:       "oauth@test.com",
		GoogleID:    ptrStr("google123"),
		PasswordHash: nil,
	}

	_, err := svc.Login(context.Background(), LoginRequest{
		Email:    "oauth@test.com",
		Password: "any",
	})
	if err != ErrOAuthOnlyAccount {
		t.Fatalf("expected ErrOAuthOnlyAccount, got %v", err)
	}
}

func TestRefresh_Success(t *testing.T) {
	svc, ur, sr := newTestAuthService(t)
	uid := uuid.New()
	ur.users["refresh@test.com"] = &domain.User{
		ID:          uid,
		Email:       "refresh@test.com",
		DisplayName: "RefreshUser",
		RunnerTier:  domain.TierFree,
	}
	sess := &domain.Session{
		ID:           uuid.New(),
		UserID:       uid,
		RefreshToken: "valid-refresh-token",
		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour),
	}
	sr.sessions = append(sr.sessions, sess)

	resp, err := svc.Refresh(context.Background(), "valid-refresh-token")
	if err != nil {
		t.Fatalf("Refresh failed: %v", err)
	}
	if resp.User.Email != "refresh@test.com" {
		t.Errorf("email = %s", resp.User.Email)
	}
	if resp.AccessToken == "" {
		t.Error("expected new access token")
	}
	if resp.RefreshToken == "" || resp.RefreshToken == "valid-refresh-token" {
		t.Error("expected new refresh token")
	}

	if sess.RevokedAt == nil {
		t.Error("expected old session to be revoked")
	}
}

func TestRefresh_InvalidToken(t *testing.T) {
	svc, _, _ := newTestAuthService(t)
	_, err := svc.Refresh(context.Background(), "nonexistent-token")
	if err != ErrInvalidToken {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestRefresh_TokenReuse(t *testing.T) {
	svc, ur, sr := newTestAuthService(t)
	uid := uuid.New()
	ur.users["reuse@test.com"] = &domain.User{
		ID:          uid,
		Email:       "reuse@test.com",
		DisplayName: "ReuseUser",
	}
	now := time.Now()
	sr.sessions = append(sr.sessions, &domain.Session{
		ID:           uuid.New(),
		UserID:       uid,
		RefreshToken: "already-revoked",
		RevokedAt:    &now,
		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour),
	})

	_, err := svc.Refresh(context.Background(), "already-revoked")
	if err != ErrTokenReused {
		t.Fatalf("expected ErrTokenReused, got %v", err)
	}
	for _, s := range sr.sessions {
		if s.UserID == uid && s.RevokedAt == nil {
			t.Error("expected all user sessions to be revoked on reuse")
		}
	}
}

func TestRefresh_UserDeletedAfterTokenIssued(t *testing.T) {
	svc, _, sr := newTestAuthService(t)
	uid := uuid.New()
	sr.sessions = append(sr.sessions, &domain.Session{
		ID:           uuid.New(),
		UserID:       uid,
		RefreshToken: "token-for-deleted-user",
		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour),
	})
	_, err := svc.Refresh(context.Background(), "token-for-deleted-user")
	if err != ErrInvalidToken {
		t.Fatalf("expected ErrInvalidToken for deleted user, got %v", err)
	}
}

func TestRegister_EmptyPassword(t *testing.T) {
	svc, _, _ := newTestAuthService(t)
	resp, err := svc.Register(context.Background(), RegisterRequest{
		Email:       "nopw@test.com",
		DisplayName: "NoPW",
	})
	if err != nil {
		t.Fatalf("Register without password failed: %v", err)
	}
	if resp.User.PasswordHash != nil {
		t.Error("expected nil password hash")
	}
}

func ptrStr(s string) *string {
	return &s
}
