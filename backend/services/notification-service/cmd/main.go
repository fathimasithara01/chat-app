package main

import (
	"log"

	"github.com/fathima-sithara/notification-service/internal/config"
	"github.com/fathima-sithara/notification-service/internal/db"
	"github.com/fathima-sithara/notification-service/internal/handler"
	"github.com/fathima-sithara/notification-service/internal/kafka"
	"github.com/fathima-sithara/notification-service/internal/repository"
	route "github.com/fathima-sithara/notification-service/internal/routes"
	"github.com/fathima-sithara/notification-service/internal/service"
	"github.com/gofiber/fiber/v2"
)

func main() {
	cfg := config.Load()

	client, err := db.Connect(cfg.MongoURI)
	if err != nil {
		log.Fatal("mongo error:", err)
	}

	dbase := client.Database(cfg.MongoDB)
	repo := repository.NewNotificationRepo(dbase)
	svc := service.New(repo)
	h := handler.New(svc)

	app := fiber.New()
	route.Register(app, h)

	go kafka.StartConsumer(cfg.KafkaBrokers, cfg.KafkaTopic, svc)

	log.Println("Notification service running on port", cfg.Port)
	log.Fatal(app.Listen(":" + cfg.Port))
}
