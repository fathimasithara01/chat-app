package route

import (
	"github.com/fathima-sithara/notification-service/internal/handler"
	"github.com/gofiber/fiber/v2"
)

func Register(app *fiber.App, h *handler.Handler) {
	api := app.Group("/api/v1/notifications")

	api.Post("/", h.SendNotification)
	api.Get("/:userID", h.GetUserNotifications)
}
