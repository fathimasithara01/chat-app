package auth

import (
	"errors"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserUUID string `json:"user_uuid"`
	jwt.RegisteredClaims
}

func ParseBearerToken(header string) (string, error) {
	if header == "" {
		return "", errors.New("authorization header empty")
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return "", errors.New("invalid authorization header format")
	}
	return parts[1], nil
}

func ParseAndValidateToken(secret, tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid token")
}
