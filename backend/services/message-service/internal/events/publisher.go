package events

import (
	"encoding/json"
	"log"

	"github.com/nats-io/nats.go"
)

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

func (p *Publisher) PublishMessageCreated(chatID string, message interface{}) {
	ev := struct {
		ChatID  string      `json:"chat_id"`
		Message interface{} `json:"message"`
	}{ChatID: chatID, Message: message}
	b, _ := json.Marshal(ev)
	if err := p.nc.Publish("message.created", b); err != nil {
		log.Println("publish message.created:", err)
	}
}
