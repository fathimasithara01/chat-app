package auth

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io/ioutil"

	"github.com/golang-jwt/jwt/v5"
)

type JWTValidator struct {
	publicKey *rsa.PublicKey
}

func NewJWTValidatorRS256(pubPath string) (*JWTValidator, error) {
	b, err := ioutil.ReadFile(pubPath)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(b)
	if block == nil {
		return nil, errors.New("invalid public key pem")
	}
	var parsed interface{}
	if block.Type == "PUBLIC KEY" {
		parsed, err = x509.ParsePKIXPublicKey(block.Bytes)
	} else {
		parsed, err = x509.ParsePKCS1PublicKey(block.Bytes)
	}
	if err != nil {
		return nil, err
	}
	rsaPub, ok := parsed.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("not rsa public key")
	}
	return &JWTValidator{publicKey: rsaPub}, nil
}

// Validate returns sub (user id) from token or error
func (j *JWTValidator) Validate(tokenStr string) (string, error) {
	tok, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		// only allow RSA
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, errors.New("invalid signing method")
		}
		return j.publicKey, nil
	})
	if err != nil {
		return "", err
	}
	if !tok.Valid {
		return "", errors.New("invalid token")
	}
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New("invalid claims")
	}
	// expecting `sub` to be user id
	if sub, ok := claims["sub"].(string); ok && sub != "" {
		return sub, nil
	}
	// fallback to user_id claim
	if uid, ok := claims["user_id"].(string); ok && uid != "" {
		return uid, nil
	}
	return "", errors.New("sub missing")
}
