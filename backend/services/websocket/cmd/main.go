package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fathima-sithara/websocket/internal/auth"
	"github.com/fathima-sithara/websocket/internal/config"
	metrics "github.com/fathima-sithara/websocket/internal/metric"
	redisclient "github.com/fathima-sithara/websocket/internal/redis"
	"github.com/fathima-sithara/websocket/internal/ws"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Init Prometheus metrics if enabled
	if cfg.EnablePrometheus {
		metrics.Init()
	}

	// Initialize Redis (Pub/Sub + Presence)
	redisclient.Init(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)

	// Load JWT public key for validating tokens
	jv, err := auth.NewJWTValidatorRS256(cfg.PublicKeyPath)
	if err != nil {
		log.Fatalf("❌ failed to load JWT validator: %v", err)
	}

	// Create WebSocket hub (with Redis pub/sub)
	hub := ws.NewHub(redisclient.Client(), cfg)
	defer hub.Shutdown()

	// Create Fiber server with WS routes
	srv := ws.NewServer(hub, jv, cfg)

	// Start server
	errChan := make(chan error, 1)
	go func() {
		addr := ":" + cfg.PortString()
		log.Printf("starting websocket service on %s", addr)
		errChan <- srv.Listen(addr)
	}()

	// Handle graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errChan:
		log.Fatalf("❌ server error: %v", err)
	case sig := <-stop:
		log.Printf("shutdown signal received: %v", sig)
	}

	// Gracefully stop Fiber
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.ShutdownWithContext(ctx); err != nil {
		log.Printf("⚠️ error shutting down server: %v", err)
	}

	// Close Redis connection pool
	_ = redisclient.Close()

	log.Println("✅ shutdown complete")
}
