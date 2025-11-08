package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fathima-sithara/chat-service/config"
	"github.com/fathima-sithara/chat-service/internal/cache"
	"github.com/fathima-sithara/chat-service/internal/kafka"
	"github.com/fathima-sithara/chat-service/internal/middleware"
	"github.com/fathima-sithara/chat-service/internal/repository"
	"github.com/fathima-sithara/chat-service/internal/routes"
	"github.com/fathima-sithara/chat-service/internal/ws"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

// Server wraps all dependencies
type Server struct {
	Cfg        *config.Config
	App        *fiber.App
	MongoRepo  *repository.MongoRepository
	Redis      *cache.Client
	KafkaProd  *kafka.Producer
	KafkaCons  *kafka.Consumer
	Hub        *ws.Hub
	JWTMw      fiber.Handler
	MsgChan    chan map[string]any
	Context    context.Context
	CancelFunc context.CancelFunc
}

func NewServer(cfg *config.Config) (*Server, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Mongo
	mongoRepo, err := repository.NewMongoRepository(cfg)
	if err != nil {
		return nil, err
	}

	// Redis
	redisClient := cache.NewRedis(cfg)

	// Kafka
	producer := kafka.NewProducer(cfg)
	consumer := kafka.NewConsumer(cfg)

	// JWT public key (only needed for verification)
	pubKey := middleware.LoadRSAPublicKey(cfg.JWTPublicKeyPath)
	jwtMw := middleware.NewJWTMiddleware(cfg.JWTPublicKeyPath)

	// WebSocket Hub
	hub := ws.NewHub(mongoRepo, producer, redisClient, pubKey)
	go hub.Run()

	// Kafka message channel
	msgChan := make(chan map[string]any, 100)

	// Fiber app
	app := fiber.New()

	s := &Server{
		Cfg:        cfg,
		App:        app,
		MongoRepo:  mongoRepo,
		Redis:      redisClient,
		KafkaProd:  producer,
		KafkaCons:  consumer,
		Hub:        hub,
		JWTMw:      jwtMw,
		MsgChan:    msgChan,
		Context:    ctx,
		CancelFunc: cancel,
	}

	return s, nil
}

// Start runs the server
func (s *Server) Start() {
	// WebSocket endpoint
	s.App.Get("/ws", ws.NewWebsocketHandler(s.Hub))

	// Routes
	routes.Register(s.App, s.Cfg, s.MongoRepo, s.KafkaProd, s.Redis, s.Hub, s.KafkaCons, s.JWTMw)

	// Kafka consumer
	go s.KafkaCons.Run(s.Context, s.MsgChan)

	// Forward Kafka messages to Hub
	go func() {
		for msg := range s.MsgChan {
			s.Hub.BroadcastJSON(msg)
		}
	}()

	// Start Fiber server
	go func() {
		log.Info().Msgf("Starting server on port %s", s.Cfg.AppPort)
		if err := s.App.Listen(":" + s.Cfg.AppPort); err != nil {
			log.Fatal().Err(err).Msg("server exited unexpectedly")
		}
	}()
}

// Shutdown gracefully closes all resources
func (s *Server) Shutdown() {
	log.Info().Msg("shutting down chat-service...")

	// Cancel context
	s.CancelFunc()

	// Close Kafka
	if err := s.KafkaCons.Close(); err != nil {
		log.Error().Err(err).Msg("failed to close Kafka consumer")
	}
	if err := s.KafkaProd.Close(); err != nil {
		log.Error().Err(err).Msg("failed to close Kafka producer")
	}

	// Close WebSocket Hub
	s.Hub.Close()

	// Close Redis
	if err := s.Redis.Close(); err != nil {
		log.Error().Err(err).Msg("failed to close Redis")
	}

	// Close Mongo
	if err := s.MongoRepo.Disconnect(context.Background()); err != nil {
		log.Error().Err(err).Msg("failed to disconnect Mongo")
	}

	// Shutdown Fiber
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.App.ShutdownWithContext(ctx); err != nil {
		log.Error().Err(err).Msg("failed to shutdown Fiber app")
	}

	log.Info().Msg("chat-service stopped gracefully")
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	server, err := NewServer(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize server")
	}

	server.Start()

	// Graceful shutdown on OS signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	server.Shutdown()
}
