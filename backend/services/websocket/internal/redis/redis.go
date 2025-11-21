package redis

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type Redis struct {
	client *redis.Client
}

func New(url string) *Redis {
	opt, _ := redis.ParseURL(url)
	client := redis.NewClient(opt)
	return &Redis{client: client}
}

func (r *Redis) Publish(ctx context.Context, channel string, msg []byte) error {
	return r.client.Publish(ctx, channel, msg).Err()
}

func (r *Redis) Subscribe(channel string) *redis.PubSub {
	return r.client.Subscribe(context.Background(), channel)
}
