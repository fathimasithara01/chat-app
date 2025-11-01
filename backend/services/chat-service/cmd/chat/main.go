package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/yourorg/chat-app/services/chat-service/internal/config"
	"github.com/yourorg/chat-app/services/chat-service/internal/handlers"
	"github.com/yourorg/chat-app/services/chat-service/internal/kafka"
	"github.com/yourorg/chat-app/services/chat-service/internal/repository"
	"github.com/yourorg/chat-app/services/chat-service/internal/service"
	"github.com/yourorg/chat-app/services/chat-service/internal/utils"
)

func main() {
	cfg, err := config.Load("./config/config.yaml")
	if err != nil {
		panic(err)
	}
	dev := cfg.App.Env == "development"
	logger, _ := utils.NewLogger(dev)
	defer logger.Sync()
	sugar := logger.Sugar()
	sugar.Infof("starting chat-service (env=%s)", cfg.App.Env)

	// Mongo
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	mc, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.Mongo.URI))
	if err != nil {
		sugar.Fatalf("mongo connect: %v", err)
	}
	if err := mc.Ping(ctx, nil); err != nil {
		sugar.Fatalf("mongo ping: %v", err)
	}
	db := mc.Database(cfg.Mongo.Database)
	msgCol := db.Collection(cfg.Mongo.MessagesCollection)
	convCol := db.Collection(cfg.Mongo.ConversationsCollection)

	// Redis
	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.Redis.Addr, Password: cfg.Redis.Password, DB: cfg.Redis.DB,
	})
	if _, err := rdb.Ping(ctx).Result(); err != nil {
		sugar.Fatalf("redis ping: %v", err)
	}

	// Kafka producer
	kp := kafka.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.TopicMessageSent)

	// repository, service, hub
	repo := repository.NewMongoRepo(msgCol, convCol)
	chatSvc := service.NewChatService(repo, kp)
	hub := websocket.NewHub()

	// handlers
	wsHandler := handlers.NewWSHandler(hub, chatSvc, cfg.App.JWTSecret)
	restHandler := handlers.NewRestHandler(chatSvc)

	// fiber app
	app := fiber.New()
	app.Use(logger.New())

	api := app.Group("/api/v1")
	api.Get("/healthz", func(c *fiber.Ctx) error { return c.SendString("ok") })
	api.Get("/conversations/:convId/messages", func(c *fiber.Ctx) error {
		// adapt chi handler to fiber: simple wrapper
		r := c.Context().Request()
		// Build minimal bridge: use restHandler directly (for brevity, call service)
		convId := c.Params("convId")
		limit := int64(50)
		msgs, err := chatSvc.GetHistory(c.Context(), convId, limit, time.Time{})
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(msgs)
	})

	// WebSocket endpoint
	app.Get("/ws", fw.New(func(c *websocket.Conn) {
		wsHandler.WS(c)
	}))

	// start server
	go func() {
		addr := fmt.Sprintf(":%d", cfg.App.Port)
		if err := app.Listen(addr); err != nil {
			sugar.Fatalf("server failed: %v", err)
		}
	}()

	// graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	sugar.Info("shutting down chat-service...")

	ctxShut, cancel2 := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel2()
	_ = app.Shutdown()
	_ = kp.Close(context.Background())
	_ = mc.Disconnect(ctxShut)
	_ = rdb.Close()
	sugar.Info("shutdown complete")
}
