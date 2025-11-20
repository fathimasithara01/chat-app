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

	"github.com/redis/go-redis/v9"
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
	db := client.Database(cfg.Mongo.DB)
	repo := repository.NewMongoRepository(db)

	rdb := redis.NewClient(&redis.Options{Addr: cfg.Redis.Addr, DB: cfg.Redis.DB})

	var jv *auth.JWTValidator
	if cfg.JWT.Algorithm == "RS256" {
		jv, err = auth.NewJWTValidatorRS256(cfg.JWT.PublicKeyPath)
	} else {
		jv, err = auth.NewJWTValidatorHS256(cfg.JWT.HSSecret)
	}
	if err != nil {
		log.Fatal("jwt:", err)
	}

	pub, err := events.NewPublisher(cfg.NATS.URL)
	if err != nil {
		log.Println("nats publisher warn:", err)
		pub = nil
	}

	sub, err := events.NewSubscriber(cfg.NATS.URL, repo)
	if err != nil {
		log.Println("nats subscriber warn:", err)
	} else {
		go sub.Start("message-service")
	}

	msgSvc := service.NewMessageService(repo, rdb)
	app := api.NewServer(cfg, msgSvc, jv, pub)

	errs := make(chan error, 1)
	go func() { errs <- app.Listen(":" + cfg.App.PortString()) }()
	log.Printf("message-service started on :%s", cfg.App.PortString())

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	select {
	case e := <-errs:
		log.Fatalf("server error: %v", e)
	case s := <-sig:
		log.Printf("signal received: %v", s)
	}

	shutdownCtx, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel2()
	if err := app.Shutdown(); err != nil {
		log.Println("fiber shutdown:", err)
	}
	_ = client.Disconnect(shutdownCtx)
	_ = rdb.Close()
	log.Println("shutdown complete")
}
