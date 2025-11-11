package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/joho/godotenv"

	"github.com/fathima-sithara/chat-service/internal/api"
	cfgpkg "github.com/fathima-sithara/chat-service/internal/config"
	"github.com/fathima-sithara/chat-service/internal/kafka"
	"github.com/fathima-sithara/chat-service/internal/repository"
	"github.com/fathima-sithara/chat-service/internal/service"
	"github.com/fathima-sithara/chat-service/internal/ws"
	"github.com/redis/go-redis/v9"
)

func main() {
	_ = godotenv.Load() // load .env if present

	cfg, err := cfgpkg.Load()
	if err != nil {
		log.Fatalf("config load: %v", err)
	}

	// Mongo client
	mc, err := repository.NewMongoClient(cfg)
	if err != nil {
		log.Fatalf("mongo init: %v", err)
	}
	defer func() { _ = mc.Disconnect(context.Background()) }()

	// Redis client
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer rdb.Close()

	// Kafka producer + consumer
	kprod := kafka.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.TopicOut)
	kcons := kafka.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.TopicIn, cfg.Kafka.GroupID)

	// Repositories & services
	repo := repository.NewMongoRepository(mc.Database(cfg.Database.Name).Collection("messages"))
	cmdSvc := service.NewCommandService(repo, rdb, kprod, cfg)
	qrySvc := service.NewQueryService(repo, rdb, cfg)

	// websocket server
	wsrv := ws.NewServer(cmdSvc, qrySvc)

	// start kafka consumer in background
	go kcons.Start(func(key string, value []byte) {
		// basic handler: forward message events to websocket hub or update status
		// keep minimal: try broadcast raw event to corresponding chat-id if present in event (message.json)
		wsrv.HandleEventMessage(key, value)
	})

	// api server
	app := api.NewServer(cfg, cmdSvc, qrySvc, wsrv)

	// run HTTP server
	go func() {
		if err := app.Listen(":" + cfg.App.PortString()); err != nil {
			log.Fatalf("server listen: %v", err)
		}
	}()
	log.Printf("chat-service started on :%s", cfg.App.PortString())

	// graceful shutdown
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
