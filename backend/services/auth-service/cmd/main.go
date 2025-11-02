package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fathima-sithara/auth-service/internal/brevo"
	"github.com/fathima-sithara/auth-service/internal/config"
	"github.com/fathima-sithara/auth-service/internal/handlers"
	"github.com/fathima-sithara/auth-service/internal/repository"
	"github.com/fathima-sithara/auth-service/internal/services"
	"github.com/fathima-sithara/auth-service/internal/twilio"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

func main() {
	// load config
	cfg, err := config.Load("config.yaml")
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}

	// logger
	var logger *zap.Logger
	if cfg.App.Env == "development" {
		l, _ := zap.NewDevelopment()
		logger = l
	} else {
		l, _ := zap.NewProduction()
		logger = l
	}
	defer logger.Sync()
	sugar := logger.Sugar()
	sugar.Infof("starting auth-service on port %d", cfg.App.Port)

	// mongo connect
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	mc, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.Mongo.URI))
	if err != nil {
		sugar.Fatalf("mongo connect: %v", err)
	}
	if err := mc.Ping(ctx, nil); err != nil {
		sugar.Fatalf("mongo ping: %v", err)
	}
	db := mc.Database(cfg.Mongo.Database)

	// redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if _, err := rdb.Ping(ctx).Result(); err != nil {
		sugar.Fatalf("redis ping: %v", err)
	}

	// clients
	tw := twilio.NewClient(cfg.Twilio.AccountSID, cfg.Twilio.AuthToken, cfg.Twilio.From)
	br := brevo.NewClient(cfg.Brevo.APIKey, cfg.Brevo.FromEmail, cfg.Brevo.FromName)

	// repo / service / handler
	userRepo := repository.NewMongoUserRepo(db, cfg.User.Collection)
	authSvc := services.NewAuthService(userRepo, tw, br, rdb, cfg.App.JWT.Secret, cfg.App.JWT.AccessTTLMinutes, cfg.App.JWT.RefreshTTLDays, cfg.Security.OtpTTLMinutes, cfg.Security.OtpRateLimitPerPhonePerHour)
	h := handlers.NewHandler(authSvc, logger)

	// fiber app
	app := fiber.New(fiber.Config{
		ReadTimeout:  cfg.App.ReadTimeout,
		WriteTimeout: cfg.App.WriteTimeout,
	})

	// simple zap logging middleware
	app.Use(func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		latency := time.Since(start)
		logger.Info("request",
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
			zap.String("ip", c.IP()),
			zap.Int("status", c.Response().StatusCode()),
			zap.Duration("latency", latency),
		)
		return err
	})

	api := app.Group("/api/v1")
	auth := api.Group("/auth")
	auth.Post("/otp/request", h.RequestOTP)
	auth.Post("/otp/verify", h.VerifyOTP)
	auth.Post("/register/email", h.RegisterEmail)
	auth.Post("/verify/email", h.VerifyEmail)
	auth.Post("/token/refresh", h.Refresh)

	// start server
	go func() {
		if err := app.Listen(fmt.Sprintf(":%d", cfg.App.Port)); err != nil {
			sugar.Fatalf("server failed: %v", err)
		}
	}()

	// graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	sugar.Info("shutting down...")
	ctxShut, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = app.Shutdown()
	_ = mc.Disconnect(ctxShut)
	_ = rdb.Close()
	sugar.Info("graceful shutdown complete")
}
