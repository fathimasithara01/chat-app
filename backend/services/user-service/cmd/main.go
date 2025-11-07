package main

import (
	"context"
	"fmt"
	stdlog "log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fathima-sithara/user-service/internal/config"
	"github.com/fathima-sithara/user-service/internal/database"
	handlers "github.com/fathima-sithara/user-service/internal/handler"
	"github.com/fathima-sithara/user-service/internal/repository"
	"github.com/fathima-sithara/user-service/internal/routes"
	"github.com/fathima-sithara/user-service/internal/service"
	"github.com/fathima-sithara/user-service/internal/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func main() {
	// Load config (config.Load uses env & YAML as implemented in your package)
	cfg, err := config.Load()
	if err != nil {
		stdlog.Fatalf("Failed to load config: %v", err)
	}
	fmt.Println("User-Service Running on Port:", cfg.App.Port)

	// Logger (zap)
	logger := utils.NewLogger(cfg.App.Env)
	defer func() {
		_ = logger.Sync()
	}()
	sugar := logger.Sugar()
	sugar.Infof("starting user-service in %s mode", cfg.App.Env)

	// Mongo URI: allow override via env
	mongoURI := cfg.MongoURI
	if v := os.Getenv("MONGO_URI"); v != "" {
		mongoURI = v
	}

	// Connect to MongoDB
	db, client, err := database.ConnectMongo(mongoURI, cfg.Mongo.Database)
	if err != nil {
		sugar.Fatalf("mongo connect failed: %v", err)
	}
	sugar.Info("connected to mongo")

	// Repository -> Service -> Handler
	userRepo := repository.NewMongoUserRepo(db, cfg.Mongo.UserCollection)
	userSvc := service.NewUserService(userRepo, os.Getenv("AUTH_SERVICE_URL"), logger)
	h := handlers.NewHandler(userSvc, logger)

	// Fiber app
	app := fiber.New(fiber.Config{
		ReadTimeout:  cfg.App.ReadTimeout,
		WriteTimeout: cfg.App.WriteTimeout,
		IdleTimeout:  cfg.App.IdleTimeout,
	})
	app.Use(cors.New())

	// Register routes
	routes.RegisterUserRoutes(app, h)

	// Start server
	go func() {
		addr := fmt.Sprintf(":%d", cfg.App.Port)
		sugar.Infof("listening on %s", addr)
		if err := app.Listen(addr); err != nil {
			sugar.Fatalf("failed to listen: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	sugar.Info("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		sugar.Errorf("fiber shutdown error: %v", err)
	}

	if err := client.Disconnect(ctx); err != nil {
		sugar.Errorf("mongo disconnect error: %v", err)
	}

	sugar.Info("graceful shutdown complete")
}
