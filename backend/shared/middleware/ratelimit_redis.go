package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	Redis  *redis.Client
	Prefix string
	Limit  int // requests
	Window time.Duration
}

func NewRateLimiter(r *redis.Client, prefix string, limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{Redis: r, Prefix: prefix, Limit: limit, Window: window}
}

func (r *RateLimiter) MiddlewareByKey(keyFunc func(c *fiber.Ctx) string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		key := keyFunc(c)
		ctx := context.Background()
		redisKey := fmt.Sprintf("%s:%s", r.Prefix, key)
		count, err := r.Redis.Incr(ctx, redisKey).Result()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "rate limiter error"})
		}
		if count == 1 {
			r.Redis.Expire(ctx, redisKey, r.Window)
		}
		if count > int64(r.Limit) {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "rate limit exceeded"})
		}
		return c.Next()
	}
}
