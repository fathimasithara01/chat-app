package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fathima-sithara/api-gateway/internal/config"
	"github.com/fathima-sithara/api-gateway/internal/discovery"
	"github.com/fathima-sithara/api-gateway/internal/middleware"
	"github.com/fathima-sithara/api-gateway/internal/proxy"
	"github.com/fathima-sithara/api-gateway/internal/router"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

func main() {
	// Logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	cfg, err := config.LoadFromEnv()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	// Service discovery (static + optional consul)
	disc, err := discovery.NewDiscovery(cfg, logger)
	if err != nil {
		logger.Fatal("discovery init failed", zap.Error(err))
	}

	// Proxy (uses discovery)
	p := proxy.NewProxy(disc, logger, cfg.CircuitBreaker)

	// Fiber app
	app := fiber.New(fiber.Config{
		Prefork:               false,
		DisableStartupMessage: true,
	})

	// Middlewares
	// JWT validates token and sets c.Locals("user_id")
	jwtMw, err := middleware.NewJWTMiddleware(cfg.JWTPublicKeyPath, logger)
	if err != nil {
		logger.Fatal("jwt middleware init failed", zap.Error(err))
	}
	rl := middleware.NewIPRateLimiter(cfg.RateLimitPerMin, logger)

	// Register routes
	router.RegisterRoutes(app, p, jwtMw, rl, logger)

	// Start server
	srvPort := cfg.Port
	go func() {
		logger.Info("starting gateway", zap.String("port", srvPort))
		if err := app.Listen(":" + srvPort); err != nil {
			logger.Fatal("server failed", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("shutting down gateway")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = app.Shutdown()
	_ = disc.Close(ctx)
	logger.Info("gateway stopped")
}
