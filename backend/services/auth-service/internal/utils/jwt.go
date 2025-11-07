package utils

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTManager handles the creation, signing, and parsing of JWTs.
type JWTManager struct {
	secret     string
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// Custom errors
var (
	ErrInvalidToken         = errors.New("invalid token")
	ErrUserIDMissing        = errors.New("user ID missing in token claims")
	ErrNotRefreshToken      = errors.New("not a refresh token")
	ErrTokenExpired         = errors.New("token is expired")
	ErrInvalidSigningMethod = errors.New("unexpected signing method")
)

// claims represents the JWT claims used for both access and refresh tokens.
type claims struct {
	UserID string `json:"sub"` // Subject (user ID)
	jwt.RegisteredClaims
}

// NewJWTManager creates and returns a new JWTManager instance.
func NewJWTManager(secret string, accessMins int, refreshDays int) *JWTManager {
	return &JWTManager{
		secret:     secret,
		accessTTL:  time.Duration(accessMins) * time.Minute,
		refreshTTL: time.Duration(refreshDays) * 24 * time.Hour,
	}
}

// GenerateAccess generates a new access token for the given userID.
func (j *JWTManager) GenerateAccess(userID string) (string, time.Time, error) {
	exp := time.Now().Add(j.accessTTL)
	claims := &claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := token.SignedString([]byte(j.secret))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign access token: %w", err)
	}
	return s, exp, nil
}

// GenerateRefresh generates a new refresh token for the given userID.
func (j *JWTManager) GenerateRefresh(userID string) (string, time.Time, error) {
	exp := time.Now().Add(j.refreshTTL)
	claims := &claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Audience:  jwt.ClaimStrings{"refresh"}, // Indicate this is a refresh token
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := token.SignedString([]byte(j.secret))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign refresh token: %w", err)
	}
	return s, exp, nil
}

// Verify parses and validates a JWT token.
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

// containsAudience checks if a specific audience string is present in the ClaimStrings slice.
func containsAudience(audiences jwt.ClaimStrings, target string) bool {
	for _, aud := range audiences {
		if aud == target {
			return true
		}
	}
	return false
}

// ParseRefresh specifically parses and validates a refresh token.
func (j *JWTManager) ParseRefresh(tokenStr string) (string, error) {
	customClaims, err := j.Verify(tokenStr)
	if err != nil {
		return "", err // Propagate validation errors
	}

	// Check if the audience claim indicates it's a refresh token
	if !containsAudience(customClaims.RegisteredClaims.Audience, "refresh") {
		return "", ErrNotRefreshToken
	}

	return customClaims.UserID, nil
}

//
func (j *JWTManager) ExtractUserID(tokenStr string) (string, error) {
	customClaims, err := j.Verify(tokenStr)
	if err != nil {
		return "", err // Propagate validation errors
	}

	// Ensure it's not a refresh token accidentally passed as an access token
	if containsAudience(customClaims.RegisteredClaims.Audience, "refresh") {
		return "", ErrNotRefreshToken // Or a more specific error for access token context
	}

	return customClaims.UserID, nil
}
