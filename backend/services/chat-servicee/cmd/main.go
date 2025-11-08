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

// Server holds service dependencies
type Server struct {
	Cfg       *config.Config
	App       *fiber.App
	MongoRepo *repository.MongoRepository
	Redis     *cache.Client
	KafkaProd *kafka.Producer
	KafkaCons *kafka.Consumer
	Hub       *ws.Hub
	JWTMw     fiber.Handler
	MsgChan   chan map[string]any

	// runtime context so we can cancel consumers etc.
	Ctx    context.Context
	Cancel context.CancelFunc
}

// NewServer builds the server and all dependencies. Errors if a required dependency fails.
func NewServer(cfg *config.Config) (*Server, error) {
	// create cancellable context for background workers (kafka consumer, etc.)
	ctx, cancel := context.WithCancel(context.Background())

	// 1) MongoDB
	mongoRepo, err := repository.NewMongoRepository(cfg)
	if err != nil {
		cancel()
		return nil, err
	}

	// 2) Redis
	redisClient := cache.NewRedis(cfg)

	// 3) Kafka producer & consumer
	producer := kafka.NewProducer(cfg)
	consumer := kafka.NewConsumer(cfg)

	pubKey := middleware.LoadRSAPublicKey(cfg.JWTPublicKeyPath)
	jwtMw := middleware.NewJWTMiddleware(cfg.JWTPublicKeyPath)

	// 5) WebSocket Hub
	hub := ws.NewHub(mongoRepo, producer, redisClient, pubKey)
	go hub.Run()

	// 6) Message channel for Kafka consumer -> broadcast
	msgChan := make(chan map[string]any, 100)

	// 7) Fiber app
	app := fiber.New()

	s := &Server{
		Cfg:       cfg,
		App:       app,
		MongoRepo: mongoRepo,
		Redis:     redisClient,
		KafkaProd: producer,
		KafkaCons: consumer,
		Hub:       hub,
		JWTMw:     jwtMw,
		MsgChan:   msgChan,
		Ctx:       ctx,
		Cancel:    cancel,
	}

	return s, nil
}

// Start wires routes, starts background workers and the HTTP server.
func (s *Server) Start() {
	// register WebSocket endpoint and API routes
	s.App.Get("/ws", ws.NewWebsocketHandler(s.Hub))
	routes.Register(s.App, s.Cfg, s.MongoRepo, s.KafkaProd, s.Redis, s.Hub, s.KafkaCons, s.JWTMw)

	// start kafka consumer (runs until context cancelled)
	go s.KafkaCons.Run(s.Ctx, s.MsgChan)

	// forward kafka messages to websocket hub
	go func() {
		for msg := range s.MsgChan {
			s.Hub.BroadcastJSON(msg)
		}
	}()

	// start HTTP server (blocking inside goroutine)
	port := s.Cfg.AppPort
	if port == "" {
		port = "8080"
	}
	go func() {
		log.Info().Msgf("starting chat-service on :%s", port)
		if err := s.App.Listen(":" + port); err != nil {
			// If Listen returns, it's fatal for this process
			log.Fatal().Err(err).Msg("fiber server exited unexpectedly")
		}
	}()
}

// Shutdown gracefully stops background workers, closes clients and shuts down the HTTP server.
func (s *Server) Shutdown() {
	log.Info().Msg("shutting down chat-service...")

	// cancel background workers (kafka consumer etc)
	s.Cancel()

	// give some time for background workers to stop
	time.Sleep(200 * time.Millisecond)

	// close kafka consumer & producer
	if s.KafkaCons != nil {
		if err := s.KafkaCons.Close(); err != nil {
			log.Error().Err(err).Msg("failed to close kafka consumer")
		}
	}
	if s.KafkaProd != nil {
		if err := s.KafkaProd.Close(); err != nil {
			log.Error().Err(err).Msg("failed to close kafka producer")
		}
	}

	// close websocket hub (disconnect clients)
	if s.Hub != nil {
		s.Hub.Close()
	}

	// close redis
	if s.Redis != nil {
		if err := s.Redis.Close(); err != nil {
			log.Error().Err(err).Msg("failed to close redis")
		}
	}

	// disconnect mongo
	if s.MongoRepo != nil {
		if err := s.MongoRepo.Disconnect(context.Background()); err != nil {
			log.Error().Err(err).Msg("failed to disconnect mongo")
		}
	}

	// shutdown fiber gracefully with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.App.ShutdownWithContext(ctx); err != nil {
		log.Error().Err(err).Msg("failed to shutdown fiber app")
	}

	log.Info().Msg("chat-service stopped gracefully")
}

func main() {
	// 1) load config
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	// 2) create server
	server, err := NewServer(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize server")
	}

	// 3) start server + background workers
	server.Start()

	// 4) Wait for OS signals and shutdown gracefully
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Info().Msgf("received signal %s, starting graceful shutdown", sig)

	server.Shutdown()
}
