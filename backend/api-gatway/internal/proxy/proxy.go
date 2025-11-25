package proxy

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/fathima-sithara/api-gateway/internal/config"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/proxy"
	"go.uber.org/zap"
)

// Proxy is a small wrapper that holds service mapping
type Proxy struct {
	services map[string]string
	log      *zap.Logger
	cfg      config.CircuitBreakerConfig
}

func NewProxyFromEnv(cfg *config.Config) (*Proxy, error) {
	if cfg == nil {
		return nil, errors.New("nil config")
	}

	p := &Proxy{
		services: cfg.Services,
		log:      zap.NewExample(),
		cfg:      cfg.CircuitBreaker,
	}

	return p, nil
}

// Lookup returns target base URL for a service name
func (p *Proxy) Lookup(service string) (string, error) {
	if t, ok := p.services[service]; ok && t != "" {
		return t, nil
	}
	return "", fmt.Errorf("service not found: %s", service)
}

// Forward returns a fiber.Handler that proxies to serviceName and strips pathPrefix
func (p *Proxy) Forward(serviceName string, pathPrefix string) (fiber.Handler, error) {
	target, err := p.Lookup(serviceName)
	if err != nil {
		return nil, err
	}

	_, err = url.Parse(target)
	if err != nil {
		return nil, err
	}

	handler := func(c *fiber.Ctx) error {
		if pathPrefix != "" {
			orig := c.OriginalURL()

			newPath := orig
			if len(orig) >= len(pathPrefix) && orig[:len(pathPrefix)] == pathPrefix {
				newPath = orig[len(pathPrefix):]
				if newPath == "" {
					newPath = "/"
				}
			}

			c.Request().SetRequestURI(newPath)
		}

		return proxy.Forward(target)(c)
	}

	return handler, nil
}

func (p *Proxy) Close(ctx context.Context) error {
	_ = ctx
	return nil
}
