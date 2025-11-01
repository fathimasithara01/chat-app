package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/segmentio/kafka-go"

	"notification-service/internal/config"
	"notification-service/internal/event_handler"
	"notification-service/internal/kafka"
	"notification-service/internal/notifier"
	"notification-service/internal/utils"
)

func main() {
	// load config
	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		log.Fatalf("config load failed: %v", err)
	}

	// logger
	dev := cfg.App.Env == "development"
	zlog, _ := utils.NewLogger(dev)
	defer zlog.Sync()
	sugar := zlog.Sugar()

	// create DLQ writer to push failed messages
	dlqWriter := kafka.NewWriter(kafka.WriterConfig{
		Brokers:      cfg.Kafka.Brokers,
		Topic:        cfg.Kafka.DLQTopic,
		RequiredAcks: kafka.RequireAll,
	})
	// NOTE: segmentio kafka.Writer imported as kafka in this file name conflict; to avoid, fully qualify:
	// but to keep code readable, alias above package import name differently if necessary.

	// initialize notifiers
	emailNotifier := notifier.NewEmailNotifier(cfg.Email.BrevoAPIKey, cfg.Email.SenderEmail, cfg.Email.SenderName, sugar)
	smsNotifier := notifier.NewSMSNotifier(cfg.SMS.TwilioSID, cfg.SMS.TwilioToken, cfg.SMS.FromPhone, sugar)

	// event handler
	handler := event_handler.NewHandler(emailNotifier, smsNotifier, dlqWriter, cfg.Kafka.MaxRetries, cfg.Kafka.RetryBackoffMs, sugar)

	// create kafka consumer
	consumer := kafka.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.TopicEvents, cfg.Kafka.GroupID, handler)

	// context & graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := consumer.Start(ctx); err != nil {
			sugar.Fatalf("consumer stopped: %v", err)
		}
	}()

	// wait for signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, os.Kill)
	<-stop
	sugar.Info("shutdown signal received, waiting for graceful stop...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer shutdownCancel()
	cancel()
	// close DLQ writer with timeout
	_ = dlqWriter.Close()
	sugar.Info("shutdown complete")
}
