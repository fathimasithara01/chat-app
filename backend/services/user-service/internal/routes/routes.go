package routes

import (
	handlers "github.com/fathima-sithara/user-service/internal/handler"
	"github.com/fathima-sithara/user-service/internal/middleware"
	"github.com/gofiber/fiber/v2"
)

func RegisterUserRoutes(app *fiber.App, h *handlers.Handler) {
	api := app.Group("/api/v1/users")

	api.Get("/me", middleware.JWT(), h.GetProfile)
	api.Put("/me", middleware.JWT(), h.UpdateProfile)
	api.Put("/change-password", middleware.JWT(), h.ChangePassword)

	api.Get("/:id", middleware.JWT(), h.GetUserByID)
	api.Delete("/:id", middleware.JWT(), h.DeleteUser)
}
