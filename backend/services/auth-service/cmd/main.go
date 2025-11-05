package main

import (
	"context"
	"fmt"
	"log" // Using standard log for early errors before zap is set up
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
	"github.com/gofiber/fiber/v2/middleware/cors" // Added CORS middleware
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables:", err) // Use log for early errors
	}

	// Load configuration
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err) // Use log.Fatalf for critical startup errors
	}

	// Initialize logger
	var logger *zap.Logger
	if cfg.App.Env == "development" {
		l, _ := zap.NewDevelopment()
		logger = l
	} else {
		l, _ := zap.NewProduction()
		logger = l
	}
	defer func() {
		_ = logger.Sync() // Flushes buffer, if any
	}()
	sugar := logger.Sugar()
	sugar.Infof("Starting auth-service in %s environment on port %d", cfg.App.Env, cfg.App.Port)

	// MongoDB Connection
	ctxMongo, cancelMongo := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelMongo()
	mc, err := mongo.Connect(ctxMongo, options.Client().ApplyURI(cfg.Mongo.URI))
	if err != nil {
		sugar.Fatalf("MongoDB connect failed: %v", err)
	}
	if err := mc.Ping(ctxMongo, nil); err != nil {
		sugar.Fatalf("MongoDB ping failed: %v", err)
	}
	db := mc.Database(cfg.Mongo.Database)
	sugar.Info("Successfully connected to MongoDB")

	// Redis Connection
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	ctxRedis, cancelRedis := context.WithTimeout(context.Background(), 5*time.Second) // Shorter timeout for redis ping
	defer cancelRedis()
	if _, err := rdb.Ping(ctxRedis).Result(); err != nil {
		sugar.Fatalf("Redis ping failed: %v", err)
	}
	sugar.Info("Successfully connected to Redis")

	// Initialize external clients
	tw := twilio.NewClient(cfg.Twilio.AccountSID, cfg.Twilio.AuthToken, cfg.Twilio.From)
	if !tw.IsConfigured() {
		sugar.Warn("Twilio client not fully configured. SMS functionality will be skipped.")
	} else {
		sugar.Info("Twilio client configured.")
	}

	br := brevo.NewClient(cfg.Brevo.APIKey, cfg.Brevo.FromEmail, cfg.Brevo.FromName)
	if !br.IsConfigured() {
		sugar.Warn("Brevo client not fully configured. Email functionality will be skipped.")
	} else {
		sugar.Info("Brevo client configured.")
	}

	// Initialize repository, service, and handler
	userRepo := repository.NewMongoUserRepo(db, cfg.User.Collection)
	authSvc := services.NewAuthService(userRepo, tw, br, rdb, cfg.App.JWT.Secret, cfg.App.JWT.AccessTTLMinutes, cfg.App.JWT.RefreshTTLDays, cfg.Security.OtpTTLMinutes, cfg.Security.OtpRateLimitPerPhonePerHour, logger)
	h := handlers.NewHandler(authSvc, logger)

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		ReadTimeout:  cfg.App.ReadTimeout,
		WriteTimeout: cfg.App.WriteTimeout,
		IdleTimeout:  cfg.App.IdleTimeout, // Added idle timeout
	})

	// Global Middlewares
	app.Use(cors.New()) // Enable CORS for development/production needs

	// Simple Zap logging middleware
	app.Use(func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		latency := time.Since(start)
		status := c.Response().StatusCode()
		if err != nil {
			// If an error occurred in a handler, Fiber might set the status code
			// but we also want to log the error itself.
			logger.Error("HTTP Request Error",
				zap.String("method", c.Method()),
				zap.String("path", c.Path()),
				zap.String("ip", c.IP()),
				zap.Int("status", status),
				zap.Duration("latency", latency),
				zap.Error(err),
			)
			return err // Re-propagate the error for Fiber's error handler
		}
		logger.Info("HTTP Request",
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
			zap.String("ip", c.IP()),
			zap.Int("status", status),
			zap.Duration("latency", latency),
		)
		return nil
	})

	// API Routes
	api := app.Group("/api/v1")
	auth := api.Group("/auth")

	// OTP based authentication
	auth.Post("/otp/request", h.RequestOTP)
	auth.Post("/otp/verify", h.VerifyOTP)

	// Email-based registration/verification (with password)
	auth.Post("/register/email", h.RegisterEmail)           // Request email OTP (for initial verification or password reset scenario)
	auth.Post("/verify/email", h.VerifyEmail)               // Verify email OTP & create/login user (without password)
	auth.Post("/register/password", h.RegisterWithPassword) // Register with email and password
	auth.Post("/login/password", h.LoginWithPassword)       // Login with email and password

	// Token management
	auth.Post("/token/refresh", h.Refresh)

	// Start server
	go func() {
		listenAddr := fmt.Sprintf(":%d", cfg.App.Port)
		sugar.Infof("Server listening on %s", listenAddr)
		if err := app.Listen(listenAddr); err != nil {
			sugar.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	sugar.Info("Shutting down server...")

	ctxShut, cancelShut := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShut()

	// Shutdown Fiber app
	if err := app.ShutdownWithContext(ctxShut); err != nil {
		sugar.Errorf("Fiber app shutdown error: %v", err)
	}

	// Disconnect MongoDB
	if err := mc.Disconnect(ctxShut); err != nil {
		sugar.Errorf("MongoDB disconnect error: %v", err)
	}

	// Close Redis connection
	if err := rdb.Close(); err != nil {
		sugar.Errorf("Redis client close error: %v", err)
	}

	sugar.Info("Graceful shutdown complete. Goodbye!")
}
