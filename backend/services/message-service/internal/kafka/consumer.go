package kafka

import (
	"context"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

type Consumer struct {
	reader *kafka.Reader
}

func NewConsumer(brokers []string, topic string, groupID string) *Consumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 1,
		MaxBytes: 10e6,
	})
	return &Consumer{reader: r}
}

func (c *Consumer) Start(handle func(key string, value []byte)) {
	ctx := context.Background()
	for {
		m, err := c.reader.ReadMessage(ctx)
		if err != nil {
			log.Printf("kafka read error: %v", err)
			time.Sleep(time.Second)
			continue
		}
		handle(string(m.Key), m.Value)
	}
}

func (c *Consumer) Close(ctx context.Context) error {
	if c.reader == nil {
		return nil
	}
	return c.reader.Close()
}
