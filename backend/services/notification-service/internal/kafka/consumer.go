package kafka

import (
	"context"
	"log"
	event_handler "notification-service/internal/handlers"
	"time"

	"github.com/segmentio/kafka-go"
)

type Consumer struct {
	reader  *kafka.Reader
	handler *event_handler.Handler
	logger  *log.Logger
}

func NewConsumer(brokers []string, topic, groupID string, handler *event_handler.Handler) *Consumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 1,
		MaxBytes: 10e6, // 10MB
	})
	return &Consumer{reader: r, handler: handler}
}

func (c *Consumer) Start(ctx context.Context) error {
	for {
		m, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			c.logger.Printf("kafka read error: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}
		// handle in goroutine to allow parallelism (bounded if needed)
		go func(msg kafka.Message) {
			_ = c.handler.HandleEvent(ctx, msg.Value)
		}(m)
	}
}
