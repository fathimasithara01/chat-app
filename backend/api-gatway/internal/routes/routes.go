package routes

import (
	"api-gateway/internal/middlewares"
	"api-gateway/internal/proxy"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// RegisterRoutes wires domain routes with proxy and middleware.
func RegisterRoutes(app *fiber.App, services map[string][]string, pubKeyPath string, logger *zap.SugaredLogger) {
	authProxy := proxy.NewServiceProxy(services["auth"])
	userProxy := proxy.NewServiceProxy(services["user"])
	chatProxy := proxy.NewServiceProxy(services["chat"])
	mediaProxy := proxy.NewServiceProxy(services["media"])
	notificationProxy := proxy.NewServiceProxy(services["notification"])

	// auth does not require JWT
	app.All("/auth/*", ProxyHandler(authProxy, logger))

	// protected routes
	app.All("/user/*", middleware.JWTAuth(pubKeyPath), ProxyHandler(userProxy, logger))
	app.All("/chat/*", middleware.JWTAuth(pubKeyPath), ProxyHandler(chatProxy, logger))
	app.All("/media/*", middleware.JWTAuth(pubKeyPath), ProxyHandler(mediaProxy, logger))
	app.All("/notification/*", middleware.JWTAuth(pubKeyPath), ProxyHandler(notificationProxy, logger))

	// health check
	app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.Status(http.StatusOK).SendString("ok")
	})
}
