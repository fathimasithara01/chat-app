package middleware

import (
	"strings"
	"time"

	"githhub.com/fathimasithara/user-service/internal/domain"
	"githhub.com/fathimasithara/user-service/internal/utils"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

func JWTMiddleware(jwtManager *utils.JWTManager, logger *zap.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing authorization header"})
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid authorization header"})
		}

		tokenStr := parts[1]
		claims, err := jwtManager.GetClaims(tokenStr)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token"})
		}

		c.Locals("user_id", claims.UserID)
		c.Locals("user_role", domain.UserRole(claims.Role))

		logger.Debug("JWT validated", zap.String("user_id", claims.UserID), zap.String("role", string(claims.Role)))
		return c.Next()
	}
}

func ZapLogger(logger *zap.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		latency := time.Since(start)

		status := c.Response().StatusCode()

		fields := []zap.Field{
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
			zap.String("ip", c.IP()),
			zap.Int("status", status),
			zap.Duration("latency", latency),
		}

		if err != nil {
			logger.Error("request failed", append(fields, zap.Error(err))...)
		} else {
			logger.Info("request completed", fields...)
		}
		return err
	}
}
