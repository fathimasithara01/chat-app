package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTManager struct {
	secret     string
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewJWTManager(secret string, accessMins int, refreshDays int) *JWTManager {
	return &JWTManager{
		secret:     secret,
		accessTTL:  time.Duration(accessMins) * time.Minute,
		refreshTTL: time.Duration(refreshDays) * 24 * time.Hour,
	}
}

func (j *JWTManager) GenerateAccess(userID string) (string, time.Time, error) {
	exp := time.Now().Add(j.accessTTL)
	claims := jwt.MapClaims{
		"sub": userID,
		"exp": exp.Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := token.SignedString([]byte(j.secret))
	return s, exp, err
}

func (j *JWTManager) GenerateRefresh(userID string) (string, time.Time, error) {
	exp := time.Now().Add(j.refreshTTL)
	claims := jwt.MapClaims{
		"sub": userID,
		"exp": exp.Unix(),
		"iat": time.Now().Unix(),
		"typ": "refresh",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := token.SignedString([]byte(j.secret))
	return s, exp, err
}

func (j *JWTManager) Verify(tokenStr string) (*jwt.Token, error) {
	return jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return []byte(j.secret), nil
	})
}

func (j *JWTManager) ExtractUserID(token *jwt.Token) (string, error) {
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", errors.New("invalid token")
	}
	sub, ok := claims["sub"].(string)
	if !ok {
		return "", errors.New("user_id missing in token")
	}
	return sub, nil
}

func (j *JWTManager) ParseRefresh(tokenStr string) (string, error) {
	token, err := j.Verify(tokenStr)
	if err != nil {
		return "", errors.New("invalid refresh token")
	}

	claims := token.Claims.(jwt.MapClaims)
	if claims["typ"] != "refresh" {
		return "", errors.New("not a refresh token")
	}

	return claims["sub"].(string), nil
}
