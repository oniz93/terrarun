package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
)

type GooglePayload struct {
	Subject string
	Email   string
	Name    string
}

func verifyGoogleIDToken(ctx context.Context, idToken, clientID string) (*GooglePayload, error) {
	url := fmt.Sprintf("https://oauth2.googleapis.com/tokeninfo?id_token=%s", idToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var claims struct {
		Sub  string `json:"sub"`
		Email string `json:"email"`
		Name string `json:"name"`
		Aud  string `json:"aud"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&claims); err != nil {
		return nil, err
	}
	if claims.Aud != clientID {
		return nil, fmt.Errorf("aud mismatch: %s", claims.Aud)
	}
	return &GooglePayload{
		Subject: claims.Sub,
		Email:   claims.Email,
		Name:    claims.Name,
	}, nil
}

type ApplePayload struct {
	Subject string
	Email   string
}

func verifyAppleIdentityToken(ctx context.Context, identityToken, teamID, keyID, privateKey string) (*ApplePayload, error) {
	token, _, err := new(jwt.Parser).ParseUnverified(identityToken, jwt.MapClaims{})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid apple token claims")
	}
	sub, _ := claims["sub"].(string)
	email, _ := claims["email"].(string)
	return &ApplePayload{
		Subject: sub,
		Email:   email,
	}, nil
}
