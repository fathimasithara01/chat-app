package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	fw "github.com/gofiber/websocket/v2"

	// "github.com/redis/go-redis/v9"
	// "go.mongodb.org/mongo-driver/mongo/options"
	// "go.uber.org/zap"

	"github.com/yourorg/chat-app/services/websocket-service/internal/config"
	"github.com/yourorg/chat-app/services/websocket-service/internal/handlers"
	"github.com/yourorg/chat-app/services/websocket-service/internal/hub"
	"github.com/yourorg/chat-app/services/websocket-service/internal/kafka"
	"github.com/yourorg/chat-app/services/websocket-service/internal/utils"
)

func main() {
	// load config
	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		panic(err)
	}
	dev := cfg.App.Env == "development"
	logger, _ := utils.NewLogger(dev)
	defer logger.Sync()
	sugar := logger.Sugar()
	sugar.Infof("starting websocket-service port=%d", cfg.App.Port)

	// Redis client
	rdb := redis2.NewClient(&redis2.Options{
		Addr: cfg.Redis.Addr, Password: cfg.Redis.Pass, DB: cfg.Redis.DB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := rdb.Ping(ctx).Result(); err != nil {
		sugar.Fatalf("redis ping failed: %v", err)
	}
	store := redispkg.NewStore(rdb, cfg.Redis.Prefix)

	// Kafka producer
	prod := kafka.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.TopicMessageSent)

	// Hub
	h := hub.NewHub()
	// set publish hook for hub to publish to Redis channel for cross-instance broadcast
	h.PublishToOtherInstances = func(ctx context.Context, channel string, payload []byte) error {
		return store.Publish(ctx, "broadcast:"+channel, payload)
	}

	// PubSub subscribe to broadcast channels for cross-instance delivery
	go func() {
		pubsub := store.Subscribe(context.Background(), "broadcast:*") // note: wildcard requires Redis KEYS or pattern subscribe, adjust below
		ch := pubsub.Channel()
		for msg := range ch {
			// msg.Channel contains actual channel like "broadcast:<convId>"
			payload := []byte(msg.Payload)
			// route by channel naming conventions
			// if channel starts with broadcast:conv:<convId> -> parse convId and broadcast locally
			// if channel starts with broadcast:user:<userUUID> -> send to user
			// simple parsing:
			// ... implement minimal example:
		}
	}()

	// create Fiber app
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		// health header or logging
		return c.Next()
	})

	// websocket route - must enable upgrade
	app.Get("/ws", fw.New(func(c *websocket.Conn) {
		// instantiate handler per connection
		hh := handlers.NewWSHandler(h, prod, store, cfg.App.JWTSecret, cfg.PingInterval, cfg.WriteDeadline, cfg.WS.MaxMessageSizeBytes, sugar)
		hh.WS(c)
	}))

	// REST health
	app.Get("/healthz", func(c *fiber.Ctx) error { return c.SendString("ok") })

	// start server
	go func() {
		addr := fmt.Sprintf(":%d", cfg.App.Port)
		if err := app.Listen(addr); err != nil {
			sugar.Fatalf("failed to listen: %v", err)
		}
	}()

	// graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	sugar.Info("shutdown signal received")
	ctxShut, cancel2 := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel2()
	_ = prod.Close(context.Background())
	_ = rdb.Close()
	_ = app.Shutdown()
	sugar.Info("websocket-service shutdown complete")
}
