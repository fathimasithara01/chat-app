package routes

import (
	"github.com/fathima-sithara/auth-service/internal/handlers"
	"github.com/gofiber/fiber/v2"
	// Import the necessary auth service or config if needed for middleware
)

// Setup configures all API routes.
func Setup(app *fiber.App, h *handlers.Handler) {
	// Middleware to inject UserID into context for protected routes (placeholder)
	authMiddleware := func(c *fiber.Ctx) error {
		// *** This is where your actual JWT validation logic would go ***
		// Example: Check Header, validate token, set c.Locals("userID", ...)
		return c.Next()
	}

	api := app.Group("/api/v1")
	auth := api.Group("/auth")

	// Public routes
	auth.Post("/register", h.Register)
	auth.Post("/verify-email", h.VerifyEmail)
	auth.Post("/login", h.Login)
	auth.Post("/request-otp", h.RequestOTP)
	auth.Post("/verify-otp", h.VerifyOTP)
	auth.Post("/refresh", h.Refresh)

	// Protected routes
	auth.Post("/logout", authMiddleware, h.Logout)
	auth.Post("/change-password", authMiddleware, h.ChangePassword)
}