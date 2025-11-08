package cache

import (
	"context"
	"time"

	"github.com/fathima-sithara/chat-service/config"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

type Client struct {
	Cli *redis.Client
}

func NewRedis(cfg *config.Config) *Client {
	r := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr, Password: cfg.RedisPassword, DB: cfg.RedisDB})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := r.Ping(ctx).Err(); err != nil {
		log.Fatal().Err(err).Msg("redis ping")
	}
	return &Client{Cli: r}
}

func (c *Client) Close() error {
	return c.Cli.Close()
}

func (c *Client) SetPresence(ctx context.Context, userID string, online bool) error {
	key := "presence:" + userID
	val := "0"
	if online {
		val = "1"
	}
	return c.Cli.Set(ctx, key, val, 0).Err()
}

func (c *Client) GetPresence(ctx context.Context, userID string) (bool, error) {
	key := "presence:" + userID
	s, err := c.Cli.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return s == "1", nil
}
