package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/fathima-sithara/message-service/internal/api"
	"github.com/fathima-sithara/message-service/internal/auth"
	"github.com/fathima-sithara/message-service/internal/config"
	"github.com/fathima-sithara/message-service/internal/kafka"
	"github.com/fathima-sithara/message-service/internal/repository"
	"github.com/fathima-sithara/message-service/internal/service"
	"github.com/fathima-sithara/message-service/internal/ws"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config load: %v", err)
	}

	mc, err := repository.NewMongoClient()
	if err != nil {
		log.Fatalf("mongo init: %v", err)
	}
	defer func() { _ = mc.Disconnect(context.Background()) }()

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer rdb.Close()

	kprod := kafka.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.TopicOut)
	kcons := kafka.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.TopicIn, cfg.Kafka.GroupID)

	repo := repository.NewMongoRepository(mc.Database(cfg.Database.Name).Collection("messages"))
	cmdSvc := service.NewCommandService(repo, rdb, kprod, cfg)
	qrySvc := service.NewQueryService(repo, rdb, cfg)

	jv, err := auth.NewJWTValidator(cfg.JWT.PublicKeyPath)
	if err != nil {
		panic(err)
	}
	wsrv := ws.NewServer(cmdSvc, qrySvc, jv)

	go kcons.Start(func(key string, value []byte) {
		wsrv.HandleEventMessage(key, value)
	})

	app := api.NewServer(cfg, cmdSvc, qrySvc, wsrv)

	go func() {
		if err := app.Listen(":" + cfg.App.PortString()); err != nil {
			log.Fatalf("server listen: %v", err)
		}
	}()
	log.Printf("chat-service started on :%s", cfg.App.PortString())

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = app.ShutdownWithContext(ctx)
	_ = kprod.Close(context.Background())
	_ = kcons.Close(context.Background())
	log.Println("chat-service stopped")
}
