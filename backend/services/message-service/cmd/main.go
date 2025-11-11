package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/fathima-sithara/message-service/internal/api"
	"github.com/fathima-sithara/message-service/internal/config"
	"github.com/fathima-sithara/message-service/internal/crypto"
	"github.com/fathima-sithara/message-service/internal/kafka"
	repo "github.com/fathima-sithara/message-service/internal/repository"
	"github.com/fathima-sithara/message-service/internal/service"
	"github.com/fathima-sithara/message-service/internal/ws"
	"github.com/redis/go-redis/v9"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// Mongo
	mc, err := repo.NewMongoClient(cfg)
	if err != nil {
		log.Fatalf("mongo init: %v", err)
	}
	defer func() { _ = mc.Disconnect(context.Background()) }()

	coll := mc.Database(cfg.Mongo.Database).Collection("messages")
	mrepo := repo.NewMongoRepository(coll)

	// Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer rdb.Close()

	// AES key
	aesKey := []byte(cfg.Security.AES256Key)

	// Kafka producer
	kprod := kafka.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.TopicIn)
	defer kprod.Close(context.Background())

	// services
	cmdSvc := service.NewCommandService(mrepo, rdb, kprod, aesKey, cfg)
	qrySvc := service.NewQueryService(mrepo, rdb, aesKey, cfg)

	// ws server
	wsSrv := ws.NewServer(cmdSvc, qrySvc)

	// jwt validator
	jwtValidator, err := crypto.NewJWTValidator("keys/public.pem")
	if err != nil {
		log.Fatalf("jwt validator: %v", err)
	}

	// api server
	app := api.NewServer(cfg, cmdSvc, qrySvc, wsSrv, jwtValidator)

	// start server
	go func() {
		addr := ":" + cfg.App.PortString()
		if err := app.Listen(addr); err != nil {
			log.Fatalf("server listen: %v", err)
		}
	}()

	log.Printf("message-service started on :%s", cfg.App.PortString())

	// graceful
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = app.ShutdownWithContext(ctx)
	_ = kprod.Close(context.Background())
	log.Println("stopped")
}
