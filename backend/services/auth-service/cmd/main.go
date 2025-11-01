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
	"go.uber.org/zap/zapcore"
)

// CustomZapLoggerMiddleware creates a Fiber middleware for Zap logging
func CustomZapLoggerMiddleware(logger *zap.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		chainErr := c.Next() // Process the request

		// Get status code and error message (if any) after request is processed
		status := c.Response().StatusCode()
		var errMsg string
		if chainErr != nil {
			errMsg = chainErr.Error()
			if e, ok := chainErr.(*fiber.Error); ok {
				status = e.Code
			}
		}

		duration := time.Since(start)

		// Log request details
		fields := []zapcore.Field{
			zap.String("ip", c.IP()),
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
			zap.Int("status", status),
			zap.String("latency", duration.String()),
			zap.String("user_agent", c.Get("User-Agent")),
			zap.String("request_id", c.Get("X-Request-ID")), // If you use request IDs
		}

		if errMsg != "" {
			fields = append(fields, zap.String("error", errMsg))
			logger.Error("Request completed with error", fields...)
		} else {
			logger.Info("Request completed", fields...)
		}

		return chainErr // Return the error from the chain
	}
}

func main() {
	cfgPath := "config.yaml"
	cfg, err := config.Load(cfgPath)
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}

	// logger
	zapCfg := zap.NewProductionConfig()
	if cfg.App.Env == "development" {
		zapCfg = zap.NewDevelopmentConfig()
	}
	logger, _ := zapCfg.Build()
	sugar := logger.Sugar()
	defer logger.Sync()

	sugar.Infof("starting auth-service on port %d", cfg.App.Port)

	// Mongo
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	mc, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.Mongo.URI))
	if err != nil {
		sugar.Fatalf("mongo connect: %v", err)
	}
	if err := mc.Ping(ctx, nil); err != nil {
		sugar.Fatalf("mongo ping: %v", err)
	}
	db := mc.Database(cfg.Mongo.Database)

	// Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if _, err := rdb.Ping(ctx).Result(); err != nil {
		sugar.Fatalf("redis ping: %v", err)
	}

	// instantiate repos & clients
	userRepo := repository.NewMongoUserRepo(db)
	tw := twilio.NewClient(cfg.Twilio.AccountSID, cfg.Twilio.AuthToken, cfg.Twilio.From)
	br := brevo.NewClient(cfg.Brevo.APIKey, cfg.Brevo.FromEmail, cfg.Brevo.FromName)

	svc := services.NewAuthService(userRepo, tw, br, rdb, cfg.App.JWT.Secret, cfg.App.JWT.AccessTTLMinutes, cfg.App.JWT.RefreshTTLDays, cfg.Security.OtpTTLMinutes, cfg.Security.OtpRateLimitPerPhonePerHour)
	h := handlers.NewHandler(svc)

	// Fiber app
	app := fiber.New(fiber.Config{
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	})

	// Apply custom Zap logger middleware
	app.Use(CustomZapLoggerMiddleware(logger))

	api := app.Group("/api/v1")
	auth := api.Group("/auth")
	auth.Post("/otp/request", h.RequestOTP)
	auth.Post("/otp/verify", h.VerifyOTP)
	auth.Post("/register/email", h.RegisterEmail)
	auth.Post("/verify/email", h.VerifyEmail)
	auth.Post("/token/refresh", h.Refresh)

	go func() {
		if err := app.Listen(fmt.Sprintf(":%d", cfg.App.Port)); err != nil {
			sugar.Fatalf("failed to start server: %v", err)
		}
	}()

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
