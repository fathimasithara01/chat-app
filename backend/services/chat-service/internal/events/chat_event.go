package events

import (
	"encoding/json"
	"log"

	"github.com/nats-io/nats.go"
)

type ChatCreatedEvent struct {
	ChatID  string   `json:"chat_id"`
	Members []string `json:"members"`
	Name    string   `json:"name"`
	IsGroup bool     `json:"is_group"`
}

type Publisher struct {
	nc *nats.Conn
}

func NewPublisher(natsURL string) (*Publisher, error) {
	nc, err := nats.Connect(natsURL)
	if err != nil {
		return nil, err
	}
	return &Publisher{nc: nc}, nil
}

func (p *Publisher) PublishChatCreated(chatID, name string, members []string, isGroup bool) error {
	event := ChatCreatedEvent{
		ChatID:  chatID,
		Name:    name,
		Members: members,
		IsGroup: isGroup,
	}
	data, _ := json.Marshal(event)
	if err := p.nc.Publish("chat.created", data); err != nil {
		log.Println("Failed to publish chat.created:", err)
		return err
	}
	return nil
}
