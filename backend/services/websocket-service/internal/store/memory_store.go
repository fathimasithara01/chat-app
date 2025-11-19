package store

import "time"

type Message struct {
	ID        string    `json:"id"`
	ChatID    string    `json:"chat_id"`
	SenderID  string    `json:"sender_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type Store interface {
	SaveMessage(m *Message) error
	GetLatestMessages(chatID string, limit int) ([]*Message, error)
}

type MemoryStore struct {
	store map[string][]*Message // chatID -> msgs
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{store: make(map[string][]*Message)}
}

func (s *MemoryStore) SaveMessage(m *Message) error {
	m.CreatedAt = time.Now().UTC()
	s.store[m.ChatID] = append(s.store[m.ChatID], m)
	// keep small: cap to 1000 per chat for memory
	if len(s.store[m.ChatID]) > 1000 {
		s.store[m.ChatID] = s.store[m.ChatID][len(s.store[m.ChatID])-1000:]
	}
	return nil
}

func (s *MemoryStore) GetLatestMessages(chatID string, limit int) ([]*Message, error) {
	msgs := s.store[chatID]
	if limit <= 0 || limit > len(msgs) {
		limit = len(msgs)
	}
	start := len(msgs) - limit
	if start < 0 {
		start = 0
	}
	out := make([]*Message, 0, limit)
	out = append(out, msgs[start:]...)
	return out, nil
}
