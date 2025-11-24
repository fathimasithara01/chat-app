package middleware

import (
	"crypto/rsa"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

var (
	publicKey     *rsa.PublicKey
	loadPublicErr error
	once          sync.Once
)

// Load RSA Public Key (only once during runtime)
func loadPublicKey(path string) (*rsa.PublicKey, error) {
	once.Do(func() {
		keyBytes, err := ioutil.ReadFile(path)
		if err != nil {
			loadPublicErr = fmt.Errorf("failed to read public key: %w", err)
			return
		}

		block, _ := pem.Decode(keyBytes)
		if block == nil {
			loadPublicErr = errors.New("invalid PEM block for public key")
			return
		}

		pub, err := jwt.ParseRSAPublicKeyFromPEM(keyBytes)
		if err != nil {
			loadPublicErr = fmt.Errorf("failed to parse RSA public key: %w", err)
			return
		}

		publicKey = pub
	})

	return publicKey, loadPublicErr
}

func JWT() fiber.Handler {
	return func(c *fiber.Ctx) error {

		// Extract token
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing Authorization header"})
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid Authorization header"})
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// Load RSA Public Key
		keyPath := os.Getenv("JWT_PUBLIC_KEY_PATH")
		if keyPath == "" {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "public key path not configured"})
		}

		pubKey, err := loadPublicKey(keyPath)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		// Parse & verify token (RS256)
		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			if t.Method.Alg() != jwt.SigningMethodRS256.Alg() {
				return nil, fmt.Errorf("unexpected jwt signing method: %v", t.Header["alg"])
			}
			return pubKey, nil
		})

		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or expired token"})
		}

		// Extract claims
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid claims"})
		}

		// Save user ID into context
		if userID, ok := claims["sub"]; ok {
			c.Locals("user_id", userID)
		} else {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing user id"})
		}

		return c.Next()
	}
}
