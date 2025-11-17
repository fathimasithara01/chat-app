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
	"github.com/fathima-sithara/message-service/internal/repository"
	"github.com/fathima-sithara/message-service/internal/service"
	"github.com/fathima-sithara/message-service/internal/ws"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("load config:", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	clientOpts := options.Client().ApplyURI(cfg.Mongo.URI)
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		log.Fatal("mongo connect:", err)
	}
	if err = client.Ping(ctx, nil); err != nil {
		log.Fatal("mongo ping:", err)
	}
	coll := client.Database(cfg.Mongo.Database).Collection("chats")

	repo := repository.NewMongoRepository(coll)
	jwtVal, err := auth.NewJWTValidator(cfg.JWT.PublicKeyPath)
	if err != nil {
		log.Fatal("jwt validator:", err)
	}

	svc := service.NewChatService(repo)
	wsSrv := ws.NewServer(svc, jwtVal)
	app := api.NewServer(cfg, svc, wsSrv, jwtVal)

	srvErr := make(chan error, 1)
	go func() {
		srvErr <- app.Listen(":" + cfg.App.PortString())
	}()

	log.Printf("Chat-Service started on %s", cfg.App.PortString())

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	select {
	case err := <-srvErr:
		log.Fatal("server error:", err)
	case s := <-sig:
		log.Printf("signal %v received, shutting down", s)
	}

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	if err := app.Shutdown(); err != nil {
		log.Printf("fiber shutdown err: %v", err)
	}
	_ = client.Disconnect(ctxShutdown)
}
