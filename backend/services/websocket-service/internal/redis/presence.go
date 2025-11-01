package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// PresenceStore stores socket mappings and presence info in Redis so other instances can route
// Keys used:
// - ws:conn:<userUUID>: set of socketIDs (or connection meta JSON)
// - ws:presence:<userUUID> -> json {status,last_seen}

type Store struct {
	client *redis.Client
	prefix string
}

type ConnMeta struct {
	ConvID   string `json:"conv_id"`
	SocketID string `json:"socket_id"`
	ConnectedAt int64 `json:"connected_at"`
}

func NewStore(r *redis.Client, prefix string) *Store {
	return &Store{client: r, prefix: prefix}
}

func (s *Store) connKey(userUUID string) string { return fmt.Sprintf("%s:conn:%s", s.prefix, userUUID) }
func (s *Store) presenceKey(userUUID string) string { return fmt.Sprintf("%s:presence:%s", s.prefix, userUUID) }

// AddConnection registers connection metadata to Redis (expires after ttl)
func (s *Store) AddConnection(ctx context.Context, userUUID, socketID, convID string, ttl time.Duration) error {
	meta := ConnMeta{ConvID: convID, SocketID: socketID, ConnectedAt: time.Now().Unix()}
	j, _ := json.Marshal(meta)
	if err := s.client.SAdd(ctx, s.connKey(userUUID), j).Err(); err != nil {
		return err
	}
	_ = s.client.Expire(ctx, s.connKey(userUUID), ttl).Err()
	// set presence
	pres := map[string]any{"status":"online","last_seen":time.Now().Unix()}
	pb, _ := json.Marshal(pres)
	return s.client.Set(ctx, s.presenceKey(userUUID), pb, ttl).Err()
}

// RemoveConnection removes a connection metadata
func (s *Store) RemoveConnection(ctx context.Context, userUUID, socketID string) error {
	// remove matching set member by scanning
	key := s.connKey(userUUID)
	members, err := s.client.SMembers(ctx, key).Result()
	if err != nil { return err }
	for _, m := range members {
		var cm ConnMeta
		_ = json.Unmarshal([]byte(m), &cm)
		if cm.SocketID == socketID {
			_ = s.client.SRem(ctx, key, m).Err()
		}
	}
	// if no members left, set presence offline
	cnt, _ := s.client.SCard(ctx, key).Result()
	if cnt == 0 {
		pres := map[string]any{"status":"offline","last_seen":time.Now().Unix()}
		pb, _ := json.Marshal(pres)
		_ = s.client.Set(ctx, s.presenceKey(userUUID), pb, 0).Err()
	}
	return nil
}

// GetPresence returns presence JSON (raw)
func (s *Store) GetPresence(ctx context.Context, userUUID string) (map[string]any, error) {
	b, err := s.client.Get(ctx, s.presenceKey(userUUID)).Bytes()
	if err != nil { return nil, err }
	var out map[string]any
	_ = json.Unmarshal(b, &out)
	return out, nil
}

// PubSub publish/subscribe to channel for cross-instance broadcast
func (s *Store) Publish(ctx context.Context, channel string, payload []byte) error {
	return s.client.Publish(ctx, channel, payload).Err()
}

func (s *Store) Subscribe(ctx context.Context, channel string) *redis.PubSub {
	return s.client.Subscribe(ctx, channel)
}
