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

// Load public key once
func loadPublicKey(path string) (*rsa.PublicKey, error) {
	once.Do(func() {
		data, err := ioutil.ReadFile(path)
		if err != nil {
			loadPublicErr = fmt.Errorf("failed to read public key: %w", err)
			return
		}
		block, _ := pem.Decode(data)
		if block == nil {
			loadPublicErr = errors.New("invalid PEM block")
			return
		}
		pub, err := jwt.ParseRSAPublicKeyFromPEM(data)
		if err != nil {
			loadPublicErr = fmt.Errorf("failed to parse public key: %w", err)
			return
		}
		publicKey = pub
	})
	return publicKey, loadPublicErr
}

// JWT middleware
func JWT() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing Authorization header"})
		}
		if !strings.HasPrefix(authHeader, "Bearer ") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid Authorization header"})
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		keyPath := os.Getenv("JWT_PUBLIC_KEY_PATH")
		if keyPath == "" {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "JWT_PUBLIC_KEY_PATH not set"})
		}

		pubKey, err := loadPublicKey(keyPath)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, errors.New("invalid signing method")
			}
			return pubKey, nil
		})
		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or expired token"})
		}

		// FIXED: your claim key is "user_id"
		sub, ok := claims["user_id"].(string)
		if !ok || sub == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing user id in token"})
		}

		c.Locals("user_id", sub)
		return c.Next()
	}
}
