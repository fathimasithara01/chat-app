package server

import (
	"time"

	"github.com/fathima-sithara/auth-service/internal/config"
	"github.com/fathima-sithara/auth-service/internal/handlers"
	"github.com/fathima-sithara/auth-service/internal/routes"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"go.uber.org/zap"
)

// New initializes the Fiber application with config, middlewares, and routes.
func New(cfg *config.Config, h *handlers.Handler, logger *zap.Logger) *fiber.App {
	app := fiber.New(fiber.Config{
		ReadTimeout:  cfg.App.ReadTimeout,
		WriteTimeout: cfg.App.WriteTimeout,
		IdleTimeout:  cfg.App.IdleTimeout,
	})

	// Global Middlewares
	app.Use(cors.New())
	app.Use(zapLoggerMiddleware(logger)) // Custom Zap logger middleware

	// Setup Routes
	routes.Setup(app, h)

	return app
}

// zapLoggerMiddleware logs incoming HTTP requests using Zap.
func zapLoggerMiddleware(logger *zap.Logger) fiber.Handler {
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
			logger.Error("HTTP Request Error", append(fields, zap.Error(err))...)
			return err
		}

		logger.Info("HTTP Request", fields...)
		return nil
	}
}
