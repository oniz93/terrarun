package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/terrarun/backend/internal/domain"
)

type JWTService struct {
	secret        string
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

func NewJWTService(secret string, accessTTL, refreshTTL time.Duration) *JWTService {
	return &JWTService{
		secret:          secret,
		accessTokenTTL:  accessTTL,
		refreshTokenTTL: refreshTTL,
	}
}

func (s *JWTService) GenerateAccessToken(user domain.User) (string, error) {
	var faction string
	if user.Faction != nil {
		faction = string(*user.Faction)
	}
	claims := jwt.MapClaims{
		"sub":     user.ID.String(),
		"email":   user.Email,
		"faction": faction,
		"tier":    string(user.RunnerTier),
		"iat":     time.Now().Unix(),
		"exp":     time.Now().Add(s.accessTokenTTL).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.secret))
}

func (s *JWTService) ParseAndValidate(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, jwt.ErrSignatureInvalid
	}
	return claims, nil
}

func (s *JWTService) GenerateRefreshToken() (string, time.Time) {
	return uuid.New().String(), time.Now().Add(s.refreshTokenTTL)
}
