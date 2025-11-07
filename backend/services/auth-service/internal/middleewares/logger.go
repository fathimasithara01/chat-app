package middlewares

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

func RequestLogger(logger *zap.SugaredLogger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		latency := time.Since(start)
		status := c.Response().StatusCode()
		if err != nil {
			logger.Errorw("HTTP Request Error",
				"method", c.Method(),
				"path", c.Path(),
				"ip", c.IP(),
				"status", status,
				"latency", latency,
				"error", err,
			)
			return err
		}
		logger.Infow("HTTP Request",
			"method", c.Method(),
			"path", c.Path(),
			"ip", c.IP(),
			"status", status,
			"latency", latency,
		)
		return nil
	}
}
