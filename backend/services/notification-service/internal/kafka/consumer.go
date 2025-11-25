package kafka

import (
	"context"
	"encoding/json"
	"log"

	"github.com/fathima-sithara/notification-service/internal/model"
	"github.com/fathima-sithara/notification-service/internal/service"
	"github.com/segmentio/kafka-go"
)

func StartConsumer(broker, topic string, svc *service.NotificationService) {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{broker},
		Topic:   topic,
		GroupID: "notification-service",
	})

	go func() {
		for {
			msg, err := r.ReadMessage(context.Background())
			if err != nil {
				log.Println("Kafka read error:", err)
				continue
			}

			var n model.Notification
			if err := json.Unmarshal(msg.Value, &n); err != nil {
				continue
			}

			svc.Send(context.Background(), &n)
		}
	}()
}
