package events

import (
	"context"
	"encoding/json"
	"log"
	"time"

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
	repo *repository.MongoRepository
}

func NewSubscriber(natsURL string, repo *repository.MongoRepository) (*Subscriber, error) {
	nc, err := nats.Connect(natsURL)
	if err != nil { return nil, err }
	return &Subscriber{nc: nc, repo: repo}, nil
}

func (s *Subscriber) Start(queue string) {
	_, err := s.nc.QueueSubscribe("chat.created", queue, func(m *nats.Msg) {
		var ev ChatCreatedEvent
		if err := json.Unmarshal(m.Data, &ev); err != nil {
			log.Println("invalid chat.created event:", err); return
		}
		for i := 0; i < 3; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			err := s.repo.InitChat(ctx, ev.ChatID, ev.Members, ev.IsGroup)
			cancel()
			if err == nil {
				log.Println("chat initialized:", ev.ChatID); break
			}
			log.Println("init chat retry err:", err)
			time.Sleep(time.Duration(i+1) * 200 * time.Millisecond)
		}
	})
	if err != nil { log.Fatal("nats subscribe error:", err) }
}
