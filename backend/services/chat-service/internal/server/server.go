package server

import (
	"context"
	"time"

	"github.com/fathima-sithara/chat-service/config"
	"github.com/fathima-sithara/chat-service/internal/cache"
	"github.com/fathima-sithara/chat-service/internal/kafka"
	"github.com/fathima-sithara/chat-service/internal/middleware"
	"github.com/fathima-sithara/chat-service/internal/repository"
	"github.com/fathima-sithara/chat-service/internal/routes"
	"github.com/fathima-sithara/chat-service/internal/ws"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

type AppServer struct {
	app *fiber.App
	cfg *config.Config
	hub *ws.Hub
}

// Listen method added to allow srv.Listen(...) in main.go
func (s *AppServer) Listen(addr string) error {
	return s.app.Listen(addr)
}

func New(cfg *config.Config, jwtMw *middleware.JWTMiddleware) (*AppServer, func(ctx context.Context) error) {
	// Initialize dependencies
	mongoRepo := repository.NewMongoRepository(cfg)
	redisClient := cache.NewRedis(cfg)
	producer := kafka.NewProducer(cfg)
	consumer := kafka.NewConsumer(cfg)
	middleware := middleware.NewJWTMiddleware(cfg.JWTAlg)

	hub := ws.NewHub(mongoRepo, producer, redisClient)
	go hub.Run()

	// Create Fiber app
	app := fiber.New()

	// WebSocket endpoint
	app.Get("/ws", websocket.New(func(c *websocket.Conn) {
		hub.HandleWebsocket(c)
	}))

	// Register HTTP routes
	routes.Register(app, cfg, mongoRepo, producer, redisClient, hub, consumer, middleware)

	server := &AppServer{
		app: app,
		cfg: cfg,
		hub: hub,
	}

	// Close function for graceful shutdown
	closeFn := func(ctx context.Context) error {
		consumer.Close()
		hub.Close()
		_ = mongoRepo.Disconnect(ctx)
		_ = redisClient.Close()

		ctx2, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		return server.app.ShutdownWithContext(ctx2)
	}

	return server, closeFn
}
