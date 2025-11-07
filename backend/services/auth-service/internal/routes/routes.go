package routes

import (
	"github.com/fathima-sithara/auth-service/internal/handlers"
	"github.com/gofiber/fiber/v2"
)

func Setup(app *fiber.App, h *handlers.Handler) {
	authMiddleware := func(c *fiber.Ctx) error {

		return c.Next()
	}

	api := app.Group("/api/v1")
	auth := api.Group("/auth")

	auth.Post("/register", h.Register)
	auth.Post("/verify-email", h.VerifyEmail)
	auth.Post("/login", h.Login)
	auth.Post("/request-otp", h.RequestOTP)
	auth.Post("/verify-otp", h.VerifyOTP)
	auth.Post("/refresh", h.Refresh)

	auth.Post("/logout", authMiddleware, h.Logout)
	auth.Post("/change-password", authMiddleware, h.ChangePassword)
}
