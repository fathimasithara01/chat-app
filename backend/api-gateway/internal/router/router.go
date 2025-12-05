package router

import (
	"net/http"

	"github.com/fathima-sithara/api-gateway/internal/middleware"
	"github.com/fathima-sithara/api-gateway/internal/proxy"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// RegisterRoutes registers gateway routes and maps them to services.
// This file uses proxy.Forward(serviceName, pathPrefix)
func RegisterRoutes(app *fiber.App, p *proxy.Proxy, jwt *middleware.JWTMiddleware, rl *middleware.IPRateLimiter, logger *zap.Logger) {
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.Status(http.StatusOK).JSON(fiber.Map{"status": "ok"})
	})

	// PUBLIC (auth)
	authHandler, err := p.Forward("auth", "/api/v1/auth")
	if err == nil {
		app.All("/api/v1/auth/*", authHandler)
	}

	// MEDIA (public example)
	if h, err := p.Forward("media", "/api/v1/media"); err == nil {
		app.All("/api/v1/media/*", h)
	}

	// Protected group - uses JWT + rate limiter
	protected := app.Group("/", jwt.Handler(), rl.Handler())

	if h, err := p.Forward("user", "/api/v1/users"); err == nil {
		protected.All("/api/v1/users/*", h)
	}

	if h, err := p.Forward("chat", "/api/v1/chat"); err == nil {
		protected.All("/api/v1/chat/*", h)
	}

	logger.Info("routes registered")
}
