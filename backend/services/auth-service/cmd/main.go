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

	ej := emailjs.NewClient(cfg.EmailJS.PublicKey, cfg.EmailJS.PrivateKey, cfg.EmailJS.ServiceID, cfg.EmailJS.TemplateID)
	if !ej.IsConfigured() {
		sugar.Warn("EmailJS client not fully configured. Email functionality will be skipped.")
	} else {
		sugar.Info("EmailJS client configured.")
	}

	userRepo := repository.NewMongoUserRepo(db, cfg.User.Collection)
	authSvc := services.NewAuthService(userRepo, tw, ej, rdb, cfg.App.JWT.Secret, cfg.App.JWT.AccessTTLMinutes, cfg.App.JWT.RefreshTTLDays, cfg.Security.OtpTTLMinutes, cfg.Security.OtpRateLimitPerPhonePerHour, logger)
	h := handlers.NewHandler(authSvc, logger)

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		ReadTimeout:  cfg.App.ReadTimeout,
		WriteTimeout: cfg.App.WriteTimeout,
		IdleTimeout:  cfg.App.IdleTimeout,
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

	// Middleware to inject UserID into context for protected routes (example)
	// You would typically have a proper JWT validation middleware here
	authMiddleware := func(c *fiber.Ctx) error {
		// This is a placeholder. In a real app, you'd extract and validate the JWT from the Authorization header.
		// For the purpose of 'Logout' and 'ChangePassword' which need a userID,
		// we'll simulate setting a userID for testing purposes or assume it's set by another middleware.
		// For now, if the token parsing from handler's Logout needs it, it can do it.
		// If you have actual JWT auth middleware, uncomment and adjust this:
		// token := c.Get("Authorization")
		// if token == "" || !strings.HasPrefix(token, "Bearer ") {
		// 	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Missing or malformed JWT"})
		// }
		// token = strings.TrimPrefix(token, "Bearer ")
		// userID, err := authSvc.GetUserIDFromAccessToken(token) // Assuming your service has this method
		// if err != nil {
		// 	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid access token"})
		// }
		// c.Locals("userID", userID)

		// For now, we'll allow routes that need userID to extract it themselves or assume a mock.
		return c.Next()
	}

	// API Routes
	api := app.Group("/api/v1")
	auth := api.Group("/auth")

	// Public routes
	auth.Post("/register", h.Register)           // Signup with email/password
	auth.Post("/verify-email", h.VerifyEmail)    // Verify email OTP & create/login user
	auth.Post("/login", h.Login)                 // Login with email and password
	auth.Post("/request-otp", h.RequestOTP)      // Request OTP for phone
	auth.Post("/verify-otp", h.VerifyOTP)        // Verify phone OTP & create/login user
	auth.Post("/refresh", h.Refresh)             // Refresh access token

	// Protected routes (require authentication, e.g., via access token)
	// Apply authMiddleware to these routes in a real application
	auth.Post("/logout", authMiddleware, h.Logout)             // Logout user
	auth.Post("/change-password", authMiddleware, h.ChangePassword) // Change password

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