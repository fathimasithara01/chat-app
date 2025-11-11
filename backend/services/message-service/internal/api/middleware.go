package api

import (
	"strings"

	"github.com/fathima-sithara/message-service/internal/crypto"
	"github.com/gofiber/fiber/v2"
)

func JWTAuthMiddleware(validator *crypto.JWTValidator) fiber.Handler {
	return func(c *fiber.Ctx) error {
		auth := c.Get("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing auth"})
		}
		token := strings.TrimPrefix(auth, "Bearer ")
		userID, err := validator.Validate(token)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token"})
		}
		c.Locals("user_id", userID)
		return c.Next()
	}
}
