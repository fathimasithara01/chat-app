package middleware

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	jwtware "github.com/gofiber/jwt/v3"
	"github.com/golang-jwt/jwt/v4"
)

// ========================== JWT MIDDLEWARE ==========================

func NewJWTMiddleware(pubKeyPath string) fiber.Handler {
	pubKey := LoadRSAPublicKey(pubKeyPath)

	return jwtware.New(jwtware.Config{
		SigningMethod: "RS256",
		SigningKey:    pubKey,
		ContextKey:    "user", // store claims => c.Locals("user")
		AuthScheme:    "Bearer",
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "unauthorized",
			})
		},
	})
}

// ========================== KEY LOADERS ==========================

// Load RSA PRIVATE KEY (PKCS1)
func LoadRSAPrivateKey(path string) *rsa.PrivateKey {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("❌ failed to read private key: %v", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		log.Fatal("❌ invalid private key: PEM decode failed")
	}

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		log.Fatalf("❌ failed parsing private key: %v", err)
	}
	return key
}

// Load RSA PUBLIC KEY (supports PKCS1 + PKIX)
func LoadRSAPublicKey(path string) *rsa.PublicKey {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("❌ failed to read public key: %v", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		log.Fatal("❌ invalid public key: PEM decode failed")
	}

	// Try PKIX first
	pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err == nil {
		if rsaPub, ok := pubKey.(*rsa.PublicKey); ok {
			return rsaPub
		}
	}

	// Fallback: RSA PUBLIC KEY format
	rsaPub, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		log.Fatalf("❌ failed parsing public key: %v", err)
	}

	return rsaPub
}

// ========================== CLAIMS ==========================

type CustomClaims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// Sign a JWT using RSA private key
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

// Verify JWT using RSA public key
func VerifyJWT(tokenStr string, publicKey *rsa.PublicKey) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return publicKey, nil
	})

	if err != nil || !token.Valid {
		return nil, err
	}

	return token.Claims.(*CustomClaims), nil
}
