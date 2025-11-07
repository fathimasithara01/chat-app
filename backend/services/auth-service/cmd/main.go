package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fathima-sithara/auth-service/internal/bootstrap"
	"github.com/fathima-sithara/auth-service/internal/server" // New package for Fiber app and middlewares
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables:", err)
	}

	// 1. Bootstrap: Load config, init logger, connect DBs, create services, and handlers
	// The cleanup function handles the deferred calls (logger.Sync, DB/Redis close)
	app, cleanup, err := bootstrap.Init()
	if err != nil {
		log.Fatalf("Application bootstrap failed: %v", err)
	}
	defer cleanup(context.Background())
	sugar := app.Sugar

	// 2. Initialize Fiber server and configure routes
	fiberApp := server.New(app.Config, app.Handler, app.Logger)

	// 3. Start server
	listenAddr := fmt.Sprintf(":%d", app.Config.App.Port)
	go func() {
		sugar.Infof("Server listening on %s", listenAddr)
		if err := fiberApp.Listen(listenAddr); err != nil {
			sugar.Fatalf("Server failed to start: %v", err)
		}
	}()

	// 4. Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	sugar.Info("Shutting down server...")

	ctxShut, cancelShut := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShut()

	// Shutdown Fiber app
	if err := fiberApp.ShutdownWithContext(ctxShut); err != nil {
		sugar.Errorf("Fiber app shutdown error: %v", err)
	}

	sugar.Info("Graceful shutdown complete. Goodbye!")
}