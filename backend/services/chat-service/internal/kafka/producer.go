package kafka

import (
	"context"
	"time"

	"github.com/fathima-sithara/chat-service/config"
	"github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(cfg *config.Config) *Producer {
	w := kafka.NewWriter(kafka.WriterConfig{Brokers: cfg.KafkaBrokers, Topic: cfg.KafkaTopicOut, Balancer: &kafka.LeastBytes{}})
	return &Producer{writer: w}
}

func (p *Producer) PublishMessage(ctx context.Context, key string, value []byte) error {
	msg := kafka.Message{Key: []byte(key), Value: value, Time: time.Now()}
	return p.writer.WriteMessages(ctx, msg)
}

func (p *Producer) Close() error { return p.writer.Close() }
