package main

import (
	"context"
	"fmt"
	"log" // Using standard log for early errors before zap is set up
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fathima-sithara/auth-service/internal/config"
	"github.com/fathima-sithara/auth-service/internal/database"
	"github.com/fathima-sithara/auth-service/internal/emailjs"
	"github.com/fathima-sithara/auth-service/internal/handlers"
	"github.com/fathima-sithara/auth-service/internal/repository"
	"github.com/fathima-sithara/auth-service/internal/services"
	"github.com/fathima-sithara/auth-service/internal/twilio"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors" // Added CORS middleware
	"github.com/joho/godotenv"
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

	// Database connections
	db, mongoClient, err := database.ConnectMongo(cfg.Mongo.URI, cfg.Mongo.Database, sugar)
	if err != nil {
		sugar.Fatal(err)
	}
	rdb, err := database.ConnectRedis(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB, sugar)
	if err != nil {
		sugar.Fatal(err)
	}

	// Initialize external clients
	tw := twilio.NewClient(cfg.Twilio.AccountSID, cfg.Twilio.AuthToken, cfg.Twilio.From)
	if !tw.IsConfigured() {
		sugar.Warn("Twilio client not fully configured. SMS functionality will be skipped.")
	} else {
		sugar.Info("Twilio client configured.")
	}

	c := emailjs.NewClient(cfg.EmailJS.PublicKey, cfg.EmailJS.PrivateKey, cfg.EmailJS.ServiceID, cfg.EmailJS.TemplateID)

	userRepo := repository.NewMongoUserRepo(db, cfg.User.Collection)
	authSvc := services.NewAuthService(userRepo, tw, c, rdb, cfg.App.JWT.Secret, cfg.App.JWT.AccessTTLMinutes, cfg.App.JWT.RefreshTTLDays, cfg.Security.OtpTTLMinutes, cfg.Security.OtpRateLimitPerPhonePerHour, logger)
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

	auth.Post("/otp/request", h.RequestOTP)
	auth.Post("/otp/verify", h.VerifyOTP)

	auth.Post("/register/email", h.RegisterEmail)           // Request email OTP (for initial verification or password reset scenario)
	auth.Post("/verify/email", h.VerifyEmail)               // Verify email OTP & create/login user (without password)
	auth.Post("/register/password", h.RegisterWithPassword) // Register with email and password
	auth.Post("/login/password", h.LoginWithPassword)       // Login with email and password

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
	if err := mongoClient.Disconnect(ctxShut); err != nil {
		sugar.Errorf("MongoDB disconnect error: %v", err)
	}

	// Close Redis connection
	if err := rdb.Close(); err != nil {
		sugar.Errorf("Redis client close error: %v", err)
	}

	sugar.Info("Graceful shutdown complete. Goodbye!")
}
