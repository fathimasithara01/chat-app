package main

import (
	"log"

	"github.com/fathima-sithara/websocket-service/internal/config"
	"github.com/fathima-sithara/websocket-service/internal/ws"
	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	cfg := config.Load()

	hub := ws.NewHub()
	server := ws.NewServer(hub)

	app := fiber.New()
	app.Get("/ws", server.HandleWS())

	log.Println("WebSocket server running on port:", cfg.Port)
	log.Fatal(app.Listen(":" + cfg.Port))
}
