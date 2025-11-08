package kafka

import (
	"context"
	"encoding/json"
	"time"

	"github.com/fathima-sithara/chat-service/config"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
)

// Broadcaster interface now passed in instead of concrete ws.Hub
type Broadcaster interface {
	BroadcastJSON(msg any)
}

type Consumer struct {
	reader *kafka.Reader
	cfg    *config.Config
}

func NewConsumer(cfg *config.Config) *Consumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: cfg.KafkaBrokers,
		Topic:   cfg.KafkaTopicIn,
		GroupID: "chat-service-group",
	})
	return &Consumer{reader: r, cfg: cfg}
}

// Run now uses the interface
func (c *Consumer) Run(b Broadcaster) {
	ctx := context.Background()
	for {
		m, err := c.reader.ReadMessage(ctx)
		if err != nil {
			log.Error().Err(err).Msg("kafka read")
			time.Sleep(1 * time.Second)
			continue
		}
		var payload map[string]any
		_ = json.Unmarshal(m.Value, &payload)
		b.BroadcastJSON(payload) // call interface method
	}
}

func (c *Consumer) Close() error { return c.reader.Close() }
