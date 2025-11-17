package events

import (
	"context"
	"encoding/json"
	"log"

	"github.com/fathima-sithara/message-service/internal/repository"
	"github.com/nats-io/nats.go"
)

type ChatCreatedEvent struct {
	ChatID  string   `json:"chat_id"`
	Members []string `json:"members"`
	Name    string   `json:"name"`
	IsGroup bool     `json:"is_group"`
}

type Subscriber struct {
	nc   *nats.Conn
	repo *repository.MessageRepository
}

func NewSubscriber(natsURL string, repo *repository.MessageRepository) (*Subscriber, error) {
	nc, err := nats.Connect(natsURL)
	if err != nil {
		return nil, err
	}
	return &Subscriber{nc: nc, repo: repo}, nil
}

func (s *Subscriber) Start() {
	_, err := s.nc.Subscribe("chat.created", func(m *nats.Msg) {
		var event ChatCreatedEvent
		if err := json.Unmarshal(m.Data, &event); err != nil {
			log.Println("Invalid event:", err)
			return
		}
		if err := s.repo.InitChat(context.Background(), event.ChatID, event.Members, event.IsGroup); err != nil {
			log.Println("Failed to init chat:", err)
		} else {
			log.Println("Chat initialized in message-service:", event.ChatID)
		}
	})
	if err != nil {
		log.Fatal("NATS subscribe error:", err)
	}
}
