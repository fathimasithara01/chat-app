package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

// LoggerMiddleware logs every request
func LoggerMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		duration := time.Since(start)

		log.Info().
			Str("method", c.Method()).
			Str("path", c.OriginalURL()).
			Int("status", c.Response().StatusCode()).
			Dur("duration", duration).
			Msg("request")

		return err
	}
}
