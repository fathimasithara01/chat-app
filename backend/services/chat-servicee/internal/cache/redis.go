package cache

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/fathima-sithara/chat-service/config"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

type Client struct {
	rdb *redis.Client
	ctx context.Context
}

// NewRedis initializes a Redis client
func NewRedis(cfg *config.Config) *Client {
	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPwd,
		DB:       cfg.RedisDB,
	})

	// Test connection
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatal().Err(err).Msg("failed to connect to redis")
	}

	log.Info().Msg("redis connected")
	return &Client{rdb: rdb, ctx: ctx}
}

// Generic key-value
func (c *Client) Set(key string, value any, expiration time.Duration) error {
	return c.rdb.Set(c.ctx, key, value, expiration).Err()
}

func (c *Client) Get(key string) (string, error) {
	return c.rdb.Get(c.ctx, key).Result()
}

func (c *Client) Delete(key string) error {
	return c.rdb.Del(c.ctx, key).Err()
}

func (c *Client) Exists(key string) (bool, error) {
	n, err := c.rdb.Exists(c.ctx, key).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// -----------------------------
// Online users
// -----------------------------
func (c *Client) MarkUserOnline(userID string) error {
	key := "online_users"
	if err := c.rdb.SAdd(c.ctx, key, userID).Err(); err != nil {
		return fmt.Errorf("MarkUserOnline failed: %w", err)
	}
	// Optionally, auto-remove user from online after inactivity
	c.rdb.Expire(c.ctx, key, 24*time.Hour)
	return nil
}

func (c *Client) MarkUserOffline(userID string) error {
	return c.rdb.SRem(c.ctx, "online_users", userID).Err()
}

func (c *Client) GetOnlineUsers() ([]string, error) {
	key := "online_users"
	users, err := c.rdb.SMembers(c.ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("GetOnlineUsers failed: %w", err)
	}
	return users, nil
}

// -----------------------------
// Typing status per chat
// -----------------------------
func (c *Client) SetTyping(chatID, userID string, isTyping bool) error {
	key := "typing:" + chatID
	if isTyping {
		return c.rdb.SAdd(c.ctx, key, userID).Err()
	} else {
		return c.rdb.SRem(c.ctx, key, userID).Err()
	}
}

func (c *Client) GetTypingUsers(chatID string) ([]string, error) {
	key := "typing:" + chatID
	users, err := c.rdb.SMembers(c.ctx, key).Result()
	if err != nil {
		return nil, err
	}
	return users, nil
}

// -----------------------------
// Last seen tracking
// -----------------------------
func (c *Client) SetLastSeen(userID string, t time.Time) error {
	key := "last_seen:" + userID
	return c.rdb.Set(c.ctx, key, t.Unix(), 0).Err()
}

func (c *Client) GetLastSeen(userID string) (time.Time, error) {
	key := "last_seen:" + userID
	val, err := c.rdb.Get(c.ctx, key).Result()
	if err != nil {
		return time.Time{}, err
	}
	ts, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(ts, 0), nil
}

// -----------------------------
// Rate limiting (messages per second)
// -----------------------------
// func (c *Client) AllowMessage(userID string, limit int, duration time.Duration) (bool, error) {
// 	key := "rate:" + userID
// 	count, err := c.rdb.Incr(c.ctx, key).Result()
// 	if err != nil {
// 		return false, err
// 	}
// 	if count == 1 {
// 		c.rdb.Expire(c.ctx, key, duration)
// 	}
// 	return count <= int64(limit), nil
// }

const luaRateLimit = `
local current = redis.call("incr", KEYS[1])
if current == 1 then
  redis.call("expire", KEYS[1], ARGV[1])
end
return current
`

func (c *Client) AllowMessage(userID string, limit int, duration time.Duration) (bool, error) {
	key := "rate:" + userID
	count, err := c.rdb.Eval(c.ctx, luaRateLimit, []string{key}, int(duration.Seconds())).Int()
	if err != nil {
		return false, err
	}
	return count <= limit, nil
}

// Close Redis client
func (c *Client) Close() error {
	return c.rdb.Close()
}
