package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fathima-sithara/websocket-service/internal/api"
	"github.com/fathima-sithara/websocket-service/internal/auth"
	"github.com/fathima-sithara/websocket-service/internal/config"
	"github.com/fathima-sithara/websocket-service/internal/store"
	"github.com/fathima-sithara/websocket-service/internal/ws"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config load: %v", err)
	}

	var jv *auth.JWTValidator
	if cfg.JWT.Algorithm == "RS256" {
		jv, err = auth.NewJWTValidatorRS256(cfg.JWT.PublicKeyPath)
	} else {
		jv, err = auth.NewJWTValidatorHS256(cfg.JWT.HSSecret)
	}
	if err != nil {
		log.Fatalf("jwt validator init: %v", err)
	}

	st := store.NewMemoryStore()
	wsSrv := ws.NewServer(jv)
	app := api.NewServer(cfg, wsSrv, st, jv)

	errs := make(chan error, 1)
	go func() {
		addr := ":" + cfg.App.PortString()
		log.Printf("starting realtime service on %s", addr)
		errs <- app.Listen(addr)
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	select {
	case e := <-errs:
		log.Fatalf("server error: %v", e)
	case s := <-sig:
		log.Printf("signal received: %v", s)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := app.Shutdown(); err != nil {
		log.Printf("fiber shutdown err: %v", err)
	}
	_ = shutdownCtx
	log.Println("shutting down")
}
