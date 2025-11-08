package middleware

import (
	"crypto/rsa"
	"io/ioutil"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog/log"
)

type JWTMiddleware struct {
	PublicKey *rsa.PublicKey
}

// Load public key from PEM
func NewJWTMiddleware(pubKeyPath string) *JWTMiddleware {
	data, err := ioutil.ReadFile(pubKeyPath)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to read JWT public key")
	}

	pubKey, err := jwt.ParseRSAPublicKeyFromPEM(data)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to parse JWT public key")
	}

	return &JWTMiddleware{PublicKey: pubKey}
}

// Fiber middleware to protect HTTP routes
func (j *JWTMiddleware) Protect() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing token"})
		}

		// Expect Bearer <token>
		tokenStr := authHeader[len("Bearer "):]

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
			if t.Method.Alg() != jwt.SigningMethodRS256.Alg() {
				return nil, fiber.NewError(fiber.StatusUnauthorized, "invalid signing method")
			}
			return j.PublicKey, nil
		})
		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token"})
		}

		claims := token.Claims.(jwt.MapClaims)
		c.Locals("user_id", claims["sub"]) // store user id for handlers
		c.Locals("email", claims["email"])

		return c.Next()
	}
}

// Validate token for WebSocket connections
func (j *JWTMiddleware) ValidateWS(tokenStr string) (map[string]any, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		return j.PublicKey, nil
	})
	if err != nil || !token.Valid {
		return nil, err
	}

	claims := token.Claims.(jwt.MapClaims)
	user := map[string]any{
		"user_id": claims["sub"],
		"email":   claims["email"],
	}
	return user, nil
}
