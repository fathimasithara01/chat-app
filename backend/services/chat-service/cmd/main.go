package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/fathima-sithara/chat-service/config"
	"github.com/fathima-sithara/chat-service/internal/middleware"
	"github.com/fathima-sithara/chat-service/internal/server"
	"github.com/rs/zerolog/log"
)

func main() {
	cfg := config.Load()
	jwtMw := middleware.NewJWTMiddleware(cfg.JWTPublicKeyPath)

	// Create server
	srv, closeFn := server.New(cfg, jwtMw)
	// Start Fiber server
	go func() {
		if err := srv.Listen(":" + cfg.AppPort); err != nil {
			log.Fatal().Err(err).Msg("server exited")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("shutting down chat-service")

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := closeFn(ctx); err != nil {
		log.Error().Err(err).Msg("failed graceful shutdown")
	}
}
