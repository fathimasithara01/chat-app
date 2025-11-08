package middleware

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	jwtware "github.com/gofiber/jwt/v3" // Fiber JWT middleware
	"github.com/golang-jwt/jwt/v4"
)

// NewJWTMiddleware creates JWT middleware using RS256 public key
func NewJWTMiddleware(pubKeyPath string) fiber.Handler {
	keyBytes, err := ioutil.ReadFile(pubKeyPath)
	if err != nil {
		log.Fatalf("failed to read jwt public key: %v", err)
	}

	pubKey, err := jwt.ParseRSAPublicKeyFromPEM(keyBytes)
	if err != nil {
		log.Fatalf("failed to parse public key: %v", err)
	}

	return jwtware.New(jwtware.Config{
		SigningMethod: "RS256",
		SigningKey:    pubKey,
		ContextKey:    "user", // JWT claims will be stored in context under "user"
		AuthScheme:    "Bearer",
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "unauthorized",
			})
		},
	})
}
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

// LoadRSAPublicKey loads an RSA public key from a PEM file
func LoadRSAPublicKey(path string) *rsa.PublicKey {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalf("failed to read public key: %v", err)
	}

	block, _ := pem.Decode(data)
	if block == nil || block.Type != "PUBLIC KEY" {
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

type CustomClaims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// SignJWT signs a token with the RSA private key
func SignJWT(userID string, privateKey *rsa.PrivateKey, expiresIn time.Duration) (string, error) {
	claims := CustomClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(privateKey)
}

// VerifyJWT verifies a token with the RSA public key
func VerifyJWT(tokenStr string, publicKey *rsa.PublicKey) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return publicKey, nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, jwt.ErrTokenInvalidClaims
}
