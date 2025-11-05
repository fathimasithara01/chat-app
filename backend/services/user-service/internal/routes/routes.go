package routes

import (
	"githhub.com/fathimasithara/user-service/internal/handler"
	"github.com/gofiber/fiber/v2"
)

func RegisterRoutes(app *fiber.App, userHandler *handler.UserHandler, jwtMiddleware fiber.Handler) {
	userHandler.RegisterRoutes(app, jwtMiddleware)
}
