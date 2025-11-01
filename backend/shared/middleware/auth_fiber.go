package middleware

import (
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	jwtv "github.com/yourorg/shared/jwt"
)

func JWTAuth(verifier *jwtv.Verifier) fiber.Handler {
	return func(c *fiber.Ctx) error {
		auth := c.Get("Authorization")
		if auth == "" {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "missing authorization"})
		}
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) != 2 {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "invalid authorization"})
		}
		token := parts[1]
		claims, err := verifier.VerifyToken(token)
		if err != nil {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token"})
		}
		c.Locals("claims", claims)
		return c.Next()
	}
}
