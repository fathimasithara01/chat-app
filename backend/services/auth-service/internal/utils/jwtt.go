package utils

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io/ioutil"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTManager struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	accessTTL  time.Duration
	refreshTTL time.Duration
}

type CustomClaims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

var (
	ErrTokenExpired = errors.New("token expired")
	ErrInvalidToken = errors.New("invalid token")
)

// Load RSA Private Key
func LoadRSAPrivateKey(path string) *rsa.PrivateKey {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalf("failed to read private key: %v", err)
	}
	block, _ := pem.Decode(data)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		log.Fatal("invalid PEM private key")
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		log.Fatalf("failed to parse private key: %v", err)
	}
	return key
}

// Load RSA Public Key
func LoadRSAPublicKey(path string) *rsa.PublicKey {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalf("failed to read public key: %v", err)
	}
	block, _ := pem.Decode(data)
	if block == nil || (block.Type != "PUBLIC KEY" && block.Type != "RSA PUBLIC KEY") {
		log.Fatal("invalid PEM public key")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		log.Fatalf("failed to parse public key: %v", err)
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		log.Fatal("not an RSA public key")
	}
	return rsaPub
}

// Create new JWT manager
func NewJWTManager(privPath, pubPath string, accessMinutes int, refreshDays int) *JWTManager {
	return &JWTManager{
		privateKey: LoadRSAPrivateKey(privPath),
		publicKey:  LoadRSAPublicKey(pubPath),
		accessTTL:  time.Duration(accessMinutes) * time.Minute,
		refreshTTL: time.Duration(refreshDays) * 24 * time.Hour,
	}
}

// Generate Access Token
func (j *JWTManager) GenerateAccessToken(userID string) (string, time.Time, error) {
	exp := time.Now().Add(j.accessTTL)
	claims := &CustomClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Audience:  jwt.ClaimStrings{"access"},
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(j.privateKey)
	return signed, exp, err
}

// Generate Refresh Token
func (j *JWTManager) GenerateRefreshToken(userID string) (string, time.Time, error) {
	exp := time.Now().Add(j.refreshTTL)
	claims := &CustomClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Audience:  jwt.ClaimStrings{"refresh"},
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(j.privateKey)
	return signed, exp, err
}

// Verify token
func (j *JWTManager) VerifyToken(tokenStr string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, ErrInvalidToken
		}
		return j.publicKey, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}
	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, ErrInvalidToken
}

// Parse Access
func (j *JWTManager) ParseAccess(tokenStr string) (string, error) {
	claims, err := j.VerifyToken(tokenStr)
	if err != nil {
		return "", err
	}
	if !containsAudience(claims.Audience, "access") {
		return "", errors.New("not an access token")
	}
	return claims.UserID, nil
}

// Parse Refresh
func (j *JWTManager) ParseRefresh(tokenStr string) (string, error) {
	claims, err := j.VerifyToken(tokenStr)
	if err != nil {
		return "", err
	}
	if !containsAudience(claims.Audience, "refresh") {
		return "", errors.New("not a refresh token")
	}
	return claims.UserID, nil
}

func containsAudience(aud jwt.ClaimStrings, target string) bool {
	for _, a := range aud {
		if a == target {
			return true
		}
	}
	return false
}
