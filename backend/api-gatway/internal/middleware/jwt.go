package middleware

import (
	"crypto/rsa"
	"errors"
	"io/ioutil"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

type JWTMiddleware struct {
	pubKey *rsa.PublicKey
	log    *zap.Logger
}

func NewJWTMiddleware(pubKeyPath string, logger *zap.Logger) (*JWTMiddleware, error) {
	data, err := ioutil.ReadFile(pubKeyPath)
	if err != nil {
		return nil, err
	}
	pub, err := jwt.ParseRSAPublicKeyFromPEM(data)
	if err != nil {
		return nil, err
	}
	return &JWTMiddleware{
		pubKey: pub,
		log:    logger,
	}, nil
}

func (j *JWTMiddleware) Handler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		auth := c.Get("Authorization")
		if auth == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing authorization"})
		}
		if !strings.HasPrefix(auth, "Bearer ") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid authorization header"})
		}
		tokenStr := strings.TrimPrefix(auth, "Bearer ")

		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, errors.New("invalid signing method")
			}
			return j.pubKey, nil
		})
		if err != nil || !token.Valid {
			j.log.Debug("jwt invalid", zap.Error(err))
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or expired token"})
		}

		// pick claim key; prefer "user_id" then "sub"
		var uid string
		if v, ok := claims["user_id"].(string); ok && v != "" {
			uid = v
		} else if v, ok := claims["sub"].(string); ok && v != "" {
			uid = v
		} else {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing user id in token"})
		}

		c.Locals("user_id", uid)
		return c.Next()
	}
}
