package utils

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTManager struct {
	secret     string
	accessTTL  time.Duration
	refreshTTL time.Duration
}

var (
	ErrNotAccessToken       = errors.New("not an access token")
	ErrInvalidToken         = errors.New("invalid token")
	ErrUserIDMissing        = errors.New("user ID missing in token claims")
	ErrNotRefreshToken      = errors.New("not a refresh token")
	ErrTokenExpired         = errors.New("token is expired")
	ErrInvalidSigningMethod = errors.New("unexpected signing method")
)

type claims struct {
	UserID string `json:"sub"`
	jwt.RegisteredClaims
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
	claims := &claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Audience:  jwt.ClaimStrings{"access"},
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := token.SignedString([]byte(j.secret))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign access token: %w", err)
	}
	return s, exp, nil
}

func (j *JWTManager) GenerateRefresh(userID string) (string, time.Time, error) {
	exp := time.Now().Add(j.refreshTTL)
	claims := &claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Audience:  jwt.ClaimStrings{"refresh"},
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := token.SignedString([]byte(j.secret))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign refresh token: %w", err)
	}
	return s, exp, nil
}

func (j *JWTManager) Verify(tokenStr string) (*claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidSigningMethod
		}
		return []byte(j.secret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) || errors.Is(err, jwt.ErrTokenNotValidYet) {
			return nil, ErrTokenExpired
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	customClaims, ok := token.Claims.(*claims)
	if !ok {
		return nil, ErrInvalidToken
	}

	if customClaims.UserID == "" {
		return nil, ErrUserIDMissing
	}

	return customClaims, nil
}

func containsAudience(audiences jwt.ClaimStrings, target string) bool {
	for _, aud := range audiences {
		if aud == target {
			return true
		}
	}
	return false
}

func (j *JWTManager) ParseRefresh(tokenStr string) (string, error) {
	customClaims, err := j.Verify(tokenStr)
	if err != nil {
		return "", err
	}

	if !containsAudience(customClaims.RegisteredClaims.Audience, "refresh") {
		return "", ErrNotRefreshToken
	}

	return customClaims.UserID, nil
}

func (j *JWTManager) ParseAccess(tokenStr string) (string, error) {
	customClaims, err := j.Verify(tokenStr)
	if err != nil {
		return "", err
	}

	if !containsAudience(customClaims.RegisteredClaims.Audience, "access") {
		return "", ErrNotAccessToken
	}

	return customClaims.UserID, nil

}

func (j *JWTManager) ExtractUserID(tokenStr string) (string, error) {
	customClaims, err := j.Verify(tokenStr)
	if err != nil {
		return "", err
	}

	return customClaims.UserID, nil
}
