package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type ctxKey string

const (
	CtxKeyUserID    ctxKey = "user_id"
	CtxKeyFaction   ctxKey = "faction"
	CtxKeyRunnerTier ctxKey = "runner_tier"
)

func UserIDFromContext(ctx context.Context) uuid.UUID {
	id, _ := ctx.Value(CtxKeyUserID).(uuid.UUID)
	return id
}

func FactionFromContext(ctx context.Context) *string {
	f, _ := ctx.Value(CtxKeyFaction).(*string)
	return f
}

func TierFromContext(ctx context.Context) string {
	t, _ := ctx.Value(CtxKeyRunnerTier).(string)
	return t
}

func Auth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearerToken(r)
			if token == "" {
				http.Error(w, `{"error":"missing authorization"}`, http.StatusUnauthorized)
				return
			}
			claims, err := parseAndValidateJWT(token, jwtSecret)
			if err != nil {
				http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
				return
			}
			userID, _ := uuid.Parse(claims["sub"].(string))
			ctx := context.WithValue(r.Context(), CtxKeyUserID, userID)
			if f, ok := claims["faction"].(string); ok {
				ctx = context.WithValue(ctx, CtxKeyFaction, &f)
			}
			if t, ok := claims["tier"].(string); ok {
				ctx = context.WithValue(ctx, CtxKeyRunnerTier, t)
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}

func parseAndValidateJWT(tokenString, secret string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, jwt.ErrSignatureInvalid
	}
	return claims, nil
}
