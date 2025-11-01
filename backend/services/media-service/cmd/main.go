package main

import (
	"context"
	"fmt"
	"media-service/internal/auth"
	"media-service/internal/config"
	"media-service/internal/handlers"
	"media-service/internal/repository"
	service "media-service/internal/services"
	"media-service/internal/storage"
	utils "media-service/internal/utis"
	"os"
	"os/signal"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// load config
	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		panic(err)
	}
	dev := cfg.App.Env == "development"

	// logger
	logger, _ := utils.NewLogger(dev)
	defer func() { _ = logger.Sync() }()

	// Mongo
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	mc, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.Mongo.URI))
	if err != nil {
		logger.Fatalf("mongo connect: %v", err)
	}
	col := mc.Database(cfg.Mongo.Database).Collection(cfg.Mongo.Collection)
	repo := repository.NewMediaRepo(col)

	// S3 store
	store, err := storage.NewS3Store(context.Background(), cfg.AWS.Region, cfg.AWS.Bucket, cfg.AWS.Endpoint, cfg.S3.PublicRead)
	if err != nil {
		logger.Fatalf("s3 init: %v", err)
	}

	// service
	presignTTL := time.Duration(cfg.S3.PresignTTL) * time.Second
	msvc := service.NewMediaService(repo, store, presignTTL)

	// JWT Verifier
	verifier, err := auth.NewJWTVerifier(cfg.JWT.PublicKeyPath)
	if err != nil {
		logger.Fatalf("jwt init: %v", err)
	}

	// fiber app & routes
	app := fiber.New(fiber.Config{
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	})
	h := handlers.NewHandler(verifier, msvc)
	app.Post("/upload", h.Upload)
	app.Get("/media/:id/url", h.GetSignedURL)
	app.Get("/healthz", func(c *fiber.Ctx) error { return c.SendString("ok") })

	// start server
	go func() {
		addr := fmt.Sprintf(":%d", cfg.App.Port)
		logger.Infof("starting media service on %s", addr)
		if err := app.Listen(addr); err != nil {
			logger.Fatalf("listen failed: %v", err)
		}
	}()

	// graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, os.Kill)
	<-quit
	logger.Info("shutdown requested")
	timeoutCtx, cancel2 := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel2()

	_ = app.Shutdown()
	_ = mc.Disconnect(timeoutCtx)
	// close other resources if any (redis, etc)
	logger.Info("shutdown completed")
}
