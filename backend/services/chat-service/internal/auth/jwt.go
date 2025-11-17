package auth

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"

	"github.com/golang-jwt/jwt/v5"
)

type JWTValidator struct {
	pub *rsa.PublicKey
}

func NewJWTValidator(path string) (*JWTValidator, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(b)
	if block == nil {
		return nil, errors.New("failed to decode public key")
	}
	pubIfc, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	pub, ok := pubIfc.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("not rsa public key")
	}
	return &JWTValidator{pub: pub}, nil
}

func (j *JWTValidator) Validate(tokenStr string) (string, error) {
	tok, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return j.pub, nil
	}, jwt.WithValidMethods([]string{"RS256"}))
	if err != nil {
		return "", err
	}
	if claims, ok := tok.Claims.(jwt.MapClaims); ok && tok.Valid {
		if sub, ok := claims["sub"].(string); ok {
			return sub, nil
		}
		if userID, ok := claims["user_id"].(string); ok {
			return userID, nil
		}
	}
	return "", errors.New("invalid token")
}
