package main

import (
	"log"

	"github.com/fathima-sithara/websocket/internal/auth"
	"github.com/fathima-sithara/websocket/internal/config"
	"github.com/fathima-sithara/websocket/internal/server"
	"github.com/fathima-sithara/websocket/internal/ws"
)

func main() {
	cfg := config.Load()

	pub, err := auth.LoadRSAPublicKey(cfg.PublicKeyPath)
	if err != nil {
		log.Fatal("failed loading public key:", err)
	}

	validator := auth.NewJWTValidator(pub)

	hub := ws.NewHub()

	go hub.Run()

	server.Start(hub, validator, cfg.Port)
}
