package kafka

import (
	"context"
	"encoding/json"
	"time"

	kafkago "github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafkago.Writer
	topic  string
}

func NewProducer(brokers []string, topic string) *Producer {
	w := &kafkago.Writer{
		Addr:         kafkago.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafkago.LeastBytes{},
		RequiredAcks: kafkago.RequireAll,
		Async:        false,
	}
	return &Producer{writer: w, topic: topic}
}

func (p *Producer) PublishMessageSent(ctx context.Context, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil { return err }
	msg := kafkago.Message{
		Key:   []byte(time.Now().Format(time.RFC3339Nano)),
		Value: b,
		Time:  time.Now(),
	}
	return p.writer.WriteMessages(ctx, msg)
}

func (p *Producer) Close(ctx context.Context) error {
	return p.writer.Close()
}
