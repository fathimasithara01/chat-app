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
	"github.com/fathima-sithara/auth-service/internal/server" 
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables:", err)
	}

	app, cleanup, err := bootstrap.Init()
	if err != nil {
		log.Fatalf("Application bootstrap failed: %v", err)
	}
	defer cleanup(context.Background())
	sugar := app.Sugar

	fiberApp := server.New(app.Config, app.Handler, app.Logger)

	listenAddr := fmt.Sprintf(":%d", app.Config.App.Port)
	go func() {
		sugar.Infof("Server listening on %s", listenAddr)
		if err := fiberApp.Listen(listenAddr); err != nil {
			sugar.Fatalf("Server failed to start: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	sugar.Info("Shutting down server...")

	ctxShut, cancelShut := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShut()

	if err := fiberApp.ShutdownWithContext(ctxShut); err != nil {
		sugar.Errorf("Fiber app shutdown error: %v", err)
	}

	sugar.Info("Graceful shutdown complete. Goodbye!")
}