package jwt

import (
	"crypto/rsa"
	"errors"
	"os"

	"github.com/golang-jwt/jwt/v5"
)

type Verifier struct {
	pub *rsa.PublicKey
}

func NewVerifier(pubKeyPath string) (*Verifier, error) {
	if pubKeyPath == "" {
		return &Verifier{pub: nil}, nil
	}
	b, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return nil, err
	}
	pub, err := jwt.ParseRSAPublicKeyFromPEM(b)
	if err != nil {
		return nil, err
	}
	return &Verifier{pub: pub}, nil
}

// VerifyToken verifies token (if public key available) and returns claims map.
func (v *Verifier) VerifyToken(tokenStr string) (jwt.MapClaims, error) {
	var token *jwt.Token
	var err error
	if v.pub != nil {
		token, err = jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			// enforce method
			if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return v.pub, nil
		})
	} else {
		// parse without validation (dev only)
		token, _, err = new(jwt.Parser).ParseUnverified(tokenStr, jwt.MapClaims{})
	}
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		return claims, nil
	}
	return nil, errors.New("invalid claims")
}

// Helper to get string claim safely
func GetStringClaim(claims jwt.MapClaims, key string) (string, bool) {
	if v, ok := claims[key]; ok {
		if s, ok := v.(string); ok {
			return s, true
		}
	}
	return "", false
}
