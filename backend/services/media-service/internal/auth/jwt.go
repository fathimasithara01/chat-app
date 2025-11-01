package auth

import (
	"crypto/rsa"
	"errors"
	"os"

	"github.com/golang-jwt/jwt/v5"
)

// JWTVerifier verifies RS256 JWT tokens and returns the user id claim (user_id or user_uuid)
type JWTVerifier struct {
	pub *rsa.PublicKey
}

func NewJWTVerifier(pubPath string) (*JWTVerifier, error) {
	b, err := os.ReadFile(pubPath)
	if err != nil {
		return nil, err
	}
	pub, err := jwt.ParseRSAPublicKeyFromPEM(b)
	if err != nil {
		return nil, err
	}
	return &JWTVerifier{pub: pub}, nil
}

// Verify returns user id (string) if valid
func (j *JWTVerifier) VerifyToken(token string) (string, error) {
	t, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return j.pub, nil
	})
	if err != nil {
		return "", err
	}
	if !t.Valid {
		return "", errors.New("invalid token")
	}
	claims, ok := t.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New("invalid claims")
	}
	// try common claim keys
	if v, ok := claims["user_id"].(string); ok {
		return v, nil
	}
	if v, ok := claims["user_uuid"].(string); ok {
		return v, nil
	}
	if v, ok := claims["sub"].(string); ok {
		return v, nil
	}
	return "", errors.New("user id not found in token")
}
