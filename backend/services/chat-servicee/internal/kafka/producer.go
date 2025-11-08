package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fathima-sithara/chat-service/config"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
	cfg    *config.Config
}

// NewProducer creates a new Kafka producer
func NewProducer(cfg *config.Config) *Producer {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.KafkaBrokers...),
		Topic:        cfg.KafkaTopicOut,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireAll,
		Async:        true, // async writes
		// ErrorLogger: kafka-go uses standard log.Logger only, so we omit or use log.Printf wrapper
	}
	return &Producer{writer: writer, cfg: cfg}
}

func (p *Producer) Publish(topic string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	msg := kafka.Message{
		Key:   []byte(time.Now().UTC().Format(time.RFC3339Nano)),
		Value: data,
		Topic: topic,
	}

	// Retry loop
	for i := 0; i < 3; i++ {
		if err := p.writer.WriteMessages(context.Background(), msg); err != nil {
			log.Warn().Err(err).Msgf("Kafka publish attempt %d failed", i+1)
			time.Sleep(500 * time.Millisecond)
			continue
		}
		return nil
	}
	return fmt.Errorf("failed to publish Kafka message after retries")
}

// Close shuts down the Kafka producer
func (p *Producer) Close() error {
	return p.writer.Close()
}
