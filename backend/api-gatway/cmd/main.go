package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fathima-sithara/api-gateway/internal/config"
	"github.com/fathima-sithara/api-gateway/internal/middleware"
	"github.com/fathima-sithara/api-gateway/internal/proxy"
	"github.com/fathima-sithara/api-gateway/internal/router"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

func main() {
	// logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// config
	cfg, err := config.LoadFromEnv()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	// load JWT middleware
	jwtMw, err := middleware.NewJWTMiddleware(cfg.JWTPublicKeyPath, logger)
	if err != nil {
		logger.Fatal("failed to init jwt middleware", zap.Error(err))
	}

	// rate limiter
	rl := middleware.NewIPRateLimiter(cfg.RateLimitPerMin, logger)

	// proxy (services map)
	prox, err := proxy.NewProxyFromEnv(cfg)
	if err != nil {
		logger.Fatal("discovery init failed", zap.Error(err))
	}

	// fiber app
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	// register routes
	router.RegisterRoutes(app, prox, jwtMw, rl, logger)

	// start server
	addr := ":" + cfg.Port
	go func() {
		logger.Info("starting gateway", zap.String("addr", addr))
		if err := app.Listen(addr); err != nil {
			logger.Fatal("server failed", zap.Error(err))
		}
	}()

	// graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("shutdown requested")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = app.Shutdown()
	_ = prox.Close(ctx)
	logger.Info("gateway stopped")
}
