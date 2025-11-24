package utils

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io/ioutil"
	"log"
	"sync"
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

var (
	privKeyOnce sync.Once
	pubKeyOnce  sync.Once
)

// Load RSA Private Key (PKCS#8)
func LoadRSAPrivateKey(path string) *rsa.PrivateKey {
	var privateKey *rsa.PrivateKey
	var loadErr error

	privKeyOnce.Do(func() {
		data, err := ioutil.ReadFile(path)
		if err != nil {
			loadErr = err
			return
		}
		block, _ := pem.Decode(data)
		if block == nil || (block.Type != "PRIVATE KEY" && block.Type != "RSA PRIVATE KEY") {
			loadErr = errors.New("invalid PEM private key")
			return
		}
		var keyInterface interface{}
		if block.Type == "PRIVATE KEY" {
			keyInterface, err = x509.ParsePKCS8PrivateKey(block.Bytes)
			if err != nil {
				loadErr = err
				return
			}
		} else {
			keyInterface, err = x509.ParsePKCS1PrivateKey(block.Bytes)
			if err != nil {
				loadErr = err
				return
			}
		}
		privateKey = keyInterface.(*rsa.PrivateKey)
	})

	if loadErr != nil {
		log.Fatalf("failed to load private key: %v", loadErr)
	}

	return privateKey
}

// Load RSA Public Key (PKIX)
func LoadRSAPublicKey(path string) *rsa.PublicKey {
	var publicKey *rsa.PublicKey
	var loadErr error

	pubKeyOnce.Do(func() {
		data, err := ioutil.ReadFile(path)
		if err != nil {
			loadErr = err
			return
		}
		block, _ := pem.Decode(data)
		if block == nil || (block.Type != "PUBLIC KEY" && block.Type != "RSA PUBLIC KEY") {
			loadErr = errors.New("invalid PEM public key")
			return
		}
		var pub interface{}
		if block.Type == "PUBLIC KEY" {
			pub, err = x509.ParsePKIXPublicKey(block.Bytes)
		} else {
			pub, err = x509.ParsePKCS1PublicKey(block.Bytes)
		}
		if err != nil {
			loadErr = err
			return
		}
		rsaPub, ok := pub.(*rsa.PublicKey)
		if !ok {
			loadErr = errors.New("not an RSA public key")
			return
		}
		publicKey = rsaPub
	})

	if loadErr != nil {
		log.Fatalf("failed to load public key: %v", loadErr)
	}

	return publicKey
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
			Subject:   userID,
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
			Subject:   userID,
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

// Parse Access Token
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

// Parse Refresh Token
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
