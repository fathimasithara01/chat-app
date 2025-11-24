package auth

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"os"

	"github.com/golang-jwt/jwt/v5"
)

type JWTValidator struct {
	alg    string
	pubKey *rsa.PublicKey
	secret []byte
}

func NewJWTValidatorRS256(pubKeyPath string) (*JWTValidator, error) {
	b, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read pubkey: %w", err)
	}
	key, err := jwt.ParseRSAPublicKeyFromPEM(b)
	if err != nil {
		return nil, fmt.Errorf("parse rsa pubkey: %w", err)
	}
	return &JWTValidator{alg: "RS256", pubKey: key}, nil
}

func NewJWTValidatorHS256(secret string) (*JWTValidator, error) {
	if secret == "" {
		return nil, errors.New("hs256 secret empty")
	}
	return &JWTValidator{alg: "HS256", secret: []byte(secret)}, nil
}

func (j *JWTValidator) Validate(token string) (string, error) {
	var keyFunc jwt.Keyfunc
	if j.alg == "RS256" {
		keyFunc = func(t *jwt.Token) (interface{}, error) {
			if t.Method.Alg() != jwt.SigningMethodRS256.Alg() {
				return nil, errors.New("unexpected signing method")
			}
			return j.pubKey, nil
		}
	} else {
		keyFunc = func(t *jwt.Token) (interface{}, error) {
			if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, errors.New("unexpected signing method")
			}
			return j.secret, nil
		}
	}

	parser := jwt.NewParser(jwt.WithValidMethods([]string{j.alg}))
	tok, err := parser.Parse(token, keyFunc)
	if err != nil {
		return "", err
	}
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok || !tok.Valid {
		return "", errors.New("invalid token")
	}
	sub, _ := claims["sub"].(string)
	if sub == "" {
		return "", errors.New("sub missing")
	}
	return sub, nil
}
