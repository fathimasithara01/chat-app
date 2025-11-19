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

	// JWT validator (supports RS256 or HS256 based on config)
	var jv *auth.JWTValidator
	if cfg.JWT.Algorithm == "RS256" {
		jv, err = auth.NewJWTValidatorRS256(cfg.JWT.PublicKeyPath)
	} else {
		jv, err = auth.NewJWTValidatorHS256(cfg.JWT.HSSecret)
	}
	if err != nil {
		log.Fatalf("jwt validator init: %v", err)
	}

	// store (in-memory for demo)
	st := store.NewMemoryStore()

	// ws server
	wsSrv := ws.NewServer(jv)

	// http server
	app := api.NewServer(cfg, wsSrv, st, jv)

	// run
	errs := make(chan error, 1)
	go func() {
		addr := ":" + cfg.App.PortString()
		log.Printf("starting realtime service on %s", addr)
		errs <- app.Listen(addr)
	}()

	// graceful shutdown
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	select {
	case e := <-errs:
		log.Fatalf("server error: %v", e)
	case s := <-sig:
		log.Printf("signal received: %v", s)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = app.Shutdown()
	log.Println("shutting down")
	_ = ctx
}
