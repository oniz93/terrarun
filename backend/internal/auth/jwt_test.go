package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/terrarun/backend/internal/domain"
)

func newTestJWT(t *testing.T) *JWTService {
	t.Helper()
	return NewJWTService("test-secret-12345", 15*time.Minute, 7*24*time.Hour)
}

func TestGenerateAndValidateAccessToken(t *testing.T) {
	svc := newTestJWT(t)
	faction := domain.FactionNeon
	user := domain.User{
		ID:         uuid.New(),
		Email:      "test@example.com",
		Faction:    &faction,
		RunnerTier: domain.TierRunner,
		DisplayName: "Tester",
	}

	token, err := svc.GenerateAccessToken(user)
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	claims, err := svc.ParseAndValidate(token)
	if err != nil {
		t.Fatalf("ParseAndValidate failed: %v", err)
	}

	if claims["sub"] != user.ID.String() {
		t.Errorf("sub = %v, want %s", claims["sub"], user.ID.String())
	}
	if claims["email"] != user.Email {
		t.Errorf("email = %v, want %s", claims["email"], user.Email)
	}
	if claims["faction"] != string(faction) {
		t.Errorf("faction = %v, want %s", claims["faction"], string(faction))
	}
	if claims["tier"] != string(domain.TierRunner) {
		t.Errorf("tier = %v, want %s", claims["tier"], string(domain.TierRunner))
	}
}

func TestParseInvalidToken(t *testing.T) {
	svc := newTestJWT(t)
	_, err := svc.ParseAndValidate("invalid-token-string")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestParseTokenWrongSecret(t *testing.T) {
	svc := newTestJWT(t)
	user := domain.User{
		ID:    uuid.New(),
		Email: "test@example.com",
	}
	token, err := svc.GenerateAccessToken(user)
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}

	otherSvc := NewJWTService("different-secret", 15*time.Minute, 7*24*time.Hour)
	_, err = otherSvc.ParseAndValidate(token)
	if err == nil {
		t.Fatal("expected error for wrong secret")
	}
}

func TestGenerateRefreshToken(t *testing.T) {
	svc := newTestJWT(t)
	token, expiresAt := svc.GenerateRefreshToken()
	if token == "" {
		t.Fatal("expected non-empty refresh token")
	}
	if expiresAt.Before(time.Now()) {
		t.Fatal("expiresAt should be in the future")
	}
	if expiresAt.After(time.Now().Add(8 * 24 * time.Hour)) {
		t.Fatal("expiresAt too far in the future (expect 7d)")
	}
}

func TestNoFactionToken(t *testing.T) {
	svc := newTestJWT(t)
	user := domain.User{
		ID:    uuid.New(),
		Email: "nofaction@example.com",
	}

	token, err := svc.GenerateAccessToken(user)
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}

	claims, err := svc.ParseAndValidate(token)
	if err != nil {
		t.Fatalf("ParseAndValidate failed: %v", err)
	}

	if claims["faction"] != "" {
		t.Errorf("expected empty faction, got %v", claims["faction"])
	}
}
