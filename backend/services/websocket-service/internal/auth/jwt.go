package auth

import (
	"errors"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type JWTValidator struct {
	secret []byte
}

func NewJWTValidator(secret string) *JWTValidator {
	return &JWTValidator{secret: []byte(secret)}
}

func (j *JWTValidator) Validate(token string) (string, error) {
	token = strings.TrimPrefix(token, "Bearer ")

	t, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		return j.secret, nil
	})
	if err != nil || !t.Valid {
		return "", errors.New("invalid token")
	}

	claims := t.Claims.(jwt.MapClaims)
	userID := claims["sub"].(string)
	return userID, nil
}
