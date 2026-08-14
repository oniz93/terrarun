package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"

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
		Sub   string `json:"sub"`
		Email string `json:"email"`
		Name  string `json:"name"`
		Aud   string `json:"aud"`
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

const appleIssuer = "https://appleid.apple.com"

// appleKey is a single JWK from Apple's public key set. Only the fields needed
// to reconstruct an RSA public key are parsed.
type appleKey struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type appleJWKS struct {
	Keys []appleKey `json:"keys"`
}

var (
	appleJWKSData  *appleJWKS
	appleJWKSFetch time.Time
	appleJWKSMu    sync.Mutex
)

// fetchAppleKeys returns Apple's signing keys, caching them for a short TTL so
// each login doesn't incur an extra network round trip.
func fetchAppleKeys(ctx context.Context) (*appleJWKS, error) {
	appleJWKSMu.Lock()
	defer appleJWKSMu.Unlock()

	if appleJWKSData != nil && time.Since(appleJWKSFetch) < 5*time.Minute {
		return appleJWKSData, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://appleid.apple.com/auth/keys", nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var jwks appleJWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, err
	}
	appleJWKSData = &jwks
	appleJWKSFetch = time.Now()
	return &jwks, nil
}

// publicKey converts the JWK modulus/exponent into an *rsa.PublicKey.
func (k appleKey) publicKey() (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, fmt.Errorf("decode apple key n: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, fmt.Errorf("decode apple key e: %w", err)
	}
	exp := 0
	for _, b := range eBytes {
		exp = exp<<8 | int(b)
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: exp}, nil
}

// verifyAppleIdentityToken verifies an Apple Sign-In identity token against
// Apple's public JWKS. It validates the RS256 signature, issuer, audience
// (when clientID is provided), and expiration (via jwt.MapClaims).
func verifyAppleIdentityToken(ctx context.Context, identityToken, clientID string) (*ApplePayload, error) {
	unverified, _, err := new(jwt.Parser).ParseUnverified(identityToken, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("parse apple token: %w", err)
	}
	kid, _ := unverified.Header["kid"].(string)

	keys, err := fetchAppleKeys(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch apple keys: %w", err)
	}

	var lastErr error
	for _, key := range keys.Keys {
		if key.Kid != kid {
			continue
		}
		pub, err := key.publicKey()
		if err != nil {
			lastErr = err
			continue
		}

		parsed, err := jwt.Parse(identityToken, func(t *jwt.Token) (interface{}, error) {
			return pub, nil
		}, jwt.WithValidMethods([]string{"RS256"}))
		if err != nil {
			lastErr = err
			continue
		}

		claims, ok := parsed.Claims.(jwt.MapClaims)
		if !ok || !parsed.Valid {
			lastErr = fmt.Errorf("invalid apple token claims")
			continue
		}

		if iss, _ := claims["iss"].(string); iss != appleIssuer {
			return nil, fmt.Errorf("invalid apple token issuer")
		}
		if clientID != "" {
			if aud, _ := claims["aud"].(string); aud != clientID {
				return nil, fmt.Errorf("invalid apple token audience")
			}
		}

		sub, _ := claims["sub"].(string)
		email, _ := claims["email"].(string)
		return &ApplePayload{Subject: sub, Email: email}, nil
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("no matching apple signing key for kid %q", kid)
}
