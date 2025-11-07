package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/fathima-sithara/user-service/internal/utils"
	"github.com/gofiber/fiber/v2"
)

func JWT() fiber.Handler {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = utils.GetJWTSecret()
	}
	return func(c *fiber.Ctx) error {
		auth := c.Get("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "missing token"})
		}
		token := strings.TrimPrefix(auth, "Bearer ")
		claims, err := utils.ParseJWT(token, secret)
		if err != nil {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token"})
		}
		if claims.UserID == "" {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token payload"})
		}
		// set user id and role
		c.Locals("user_id", claims.UserID)
		c.Locals("role", claims.Role)
		return c.Next()
	}
}
