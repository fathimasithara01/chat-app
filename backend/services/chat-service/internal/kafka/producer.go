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
	w := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireAll,
		Async:        false,
	}
	return &Producer{writer: w, topic: topic}
}

func (p *Producer) PublishMessageSent(ctx context.Context, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	msg := kafka.Message{
		Key:   []byte(time.Now().Format(time.RFC3339Nano)),
		Value: b,
	}
	return p.writer.WriteMessages(ctx, msg)
}

func (p *Producer) Close(ctx context.Context) error {
	return p.writer.Close()
}
