package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"api-gateway/internal/config"
	"api-gateway/internal/middlewares"
	"api-gateway/internal/routes"
	"api-gateway/internal/utils"

	"github.com/gofiber/fiber/v2"
	flogger "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	// load config
	cfg, err := config.LoadConfig("config/config.yaml")
	if err != nil {
		panic(err)
	}

	// logger
	z, _ := utils.NewZapLogger(cfg.Log.Level == "debug")
	defer z.Sync()
	sugar := z.Sugar()
	sugar.Infof("starting api-gateway on port %s", cfg.Server.Port)

	// fiber app
	app := fiber.New(fiber.Config{
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeoutSeconds) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeoutSeconds) * time.Second,
	})

	// middlewares
	app.Use(recover.New())
	app.Use(flogger.New())
	app.Use(middleware.RequestID())

	// register routes
	routes.RegisterRoutes(app, cfg.Services, cfg.JWT.PublicKeyPath, sugar)

	// start server
	go func() {
		addr := fmt.Sprintf(":%s", cfg.Server.Port)
		if err := app.Listen(addr); err != nil {
			sugar.Fatalf("server failed: %v", err)
		}
	}()

	// graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	sugar.Info("shutdown signal received, shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = app.Shutdown()
	sugar.Info("api-gateway stopped")
	_ = ctx
}
