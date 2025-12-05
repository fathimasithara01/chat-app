package middleware

import (
	"net"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

type IPRateLimiter struct {
	visitors sync.Map
	rps      rate.Limit
	burst    int
	log      *zap.Logger
}

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func NewIPRateLimiter(perMinute int, logger *zap.Logger) *IPRateLimiter {
	rps := rate.Limit(float64(perMinute) / 60.0)
	l := &IPRateLimiter{
		visitors: sync.Map{},
		rps:      rps,
		burst:    5,
		log:      logger,
	}
	go l.cleanupVisitors()
	return l
}

func (l *IPRateLimiter) getLimiter(ip string) *rate.Limiter {
	v, ok := l.visitors.Load(ip)
	if ok {
		vi := v.(*visitor)
		vi.lastSeen = time.Now()
		return vi.limiter
	}
	lim := rate.NewLimiter(l.rps, l.burst)
	l.visitors.Store(ip, &visitor{limiter: lim, lastSeen: time.Now()})
	return lim
}

func (l *IPRateLimiter) cleanupVisitors() {
	for {
		time.Sleep(time.Minute)
		cutoff := time.Now().Add(-5 * time.Minute)
		l.visitors.Range(func(k, v interface{}) bool {
			vi := v.(*visitor)
			if vi.lastSeen.Before(cutoff) {
				l.visitors.Delete(k)
			}
			return true
		})
	}
}

func (l *IPRateLimiter) Handler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ip := getIP(c)
		limiter := l.getLimiter(ip)
		if !limiter.Allow() {
			l.log.Warn("rate limit exceeded", zap.String("ip", ip), zap.String("path", c.Path()))
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "rate limit exceeded"})
		}
		return c.Next()
	}
}

func getIP(c *fiber.Ctx) string {
	ip := c.IP()
	if ip == "" {
		ip = "unknown"
	}
	host, _, err := net.SplitHostPort(ip)
	if err == nil {
		return host
	}
	return ip
}
