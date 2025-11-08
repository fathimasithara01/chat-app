package kafka

import (
	"context"
	"encoding/json"
	"time"

	"github.com/fathima-sithara/chat-service/config"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
)

// Consumer reads messages from Kafka
type Consumer struct {
	reader *kafka.Reader
	cfg    *config.Config
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(cfg *config.Config) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  cfg.KafkaBrokers,
		GroupID:  "chat-service-group",
		Topic:    cfg.KafkaTopicIn,
		MinBytes: 1,
		MaxBytes: 10e6,
	})
	return &Consumer{reader: reader, cfg: cfg}
}

func (c *Consumer) Run(ctx context.Context, msgChan chan<- map[string]any) {
	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("Kafka consumer stopping")
			return
		default:
			msg, err := c.reader.ReadMessage(ctx)
			if err != nil {
				log.Error().Err(err).Msg("kafka read error")
				time.Sleep(time.Second)
				continue
			}

			var payload map[string]any
			if err := json.Unmarshal(msg.Value, &payload); err != nil {
				log.Error().Err(err).Msg("failed to unmarshal kafka message")
				continue
			}
			msgChan <- payload
		}
	}
}

// Close stops the consumer
func (c *Consumer) Close() error {
	return c.reader.Close()
}
