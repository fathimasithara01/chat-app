package kafka

import (
	"context"
	"encoding/json"
	"time"

	"github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
	topic  string
}

func NewProducer(brokers []string, topic string) *Producer {
	w := kafka.NewWriter(kafka.WriterConfig{
		Brokers:  brokers,
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	})
	return &Producer{writer: w, topic: topic}
}

func (p *Producer) PublishMessage(ctx context.Context, key string, v interface{}) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	msg := kafka.Message{
		Key:   []byte(key),
		Value: b,
		Time:  time.Now(),
	}
	return p.writer.WriteMessages(ctx, msg)
}

func (p *Producer) Close(ctx context.Context) error {
	if p.writer == nil {
		return nil
	}
	return p.writer.Close()
}
