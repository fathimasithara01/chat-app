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

type Publisher struct{ nc *nats.Conn }

func NewPublisher(url string) (*Publisher, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, err
	}
	return &Publisher{nc: nc}, nil
}

func (p *Publisher) PublishChatCreated(chatID, name string, members []string, isGroup bool) error {
	if p == nil || p.nc == nil { return nil }
	ev := ChatCreatedEvent{ChatID: chatID, Name: name, Members: members, IsGroup: isGroup}
	b, _ := json.Marshal(ev)
	if err := p.nc.Publish("chat.created", b); err != nil {
		log.Println("publish chat.created:", err)
		return err
	}
	return nil
}
