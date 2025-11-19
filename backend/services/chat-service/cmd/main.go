package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fathima-sithara/message-service/internal/api"
	"github.com/fathima-sithara/message-service/internal/auth"
	"github.com/fathima-sithara/message-service/internal/config"
	"github.com/fathima-sithara/message-service/internal/events"
	"github.com/fathima-sithara/message-service/internal/repository"
	"github.com/fathima-sithara/message-service/internal/service"
	"github.com/fathima-sithara/message-service/internal/ws"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("config:", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.Mongo.URI))
	if err != nil {
		log.Fatal("mongo connect:", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatal("mongo ping:", err)
	}

	coll := client.Database(cfg.Mongo.Database).Collection("chats")
	repo := repository.NewMongoRepository(coll)

	jv, err := auth.NewJWTValidator(cfg.JWT.PublicKeyPath, cfg.JWT.Algorithm, cfg.JWT.Secret)
	if err != nil {
		log.Fatal("jwt:", err)
	}

	pub, err := events.NewPublisher(cfg.NATS.URL)
	if err != nil {
		log.Println("nats warn:", err)
		pub = nil
	}

	svc := service.NewChatService(repo, pub)
	wsSrv := ws.NewServer(svc, jv)
	app := api.NewServer(cfg, svc, wsSrv, jv)

	errs := make(chan error, 1)
	go func() { errs <- app.Listen(":" + cfg.App.PortString()) }()

	log.Printf("started on :%s", cfg.App.PortString())

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	select {
	case e := <-errs:
		log.Fatal("server error:", e)
	case s := <-sig:
		log.Printf("signal %v received", s)
	}

	shutdownCtx, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel2()
	if err := app.Shutdown(); err != nil {
		log.Println("fiber shutdown:", err)
	}
	if err := client.Disconnect(shutdownCtx); err != nil {
		log.Println("mongo disconnect:", err)
	}
}
