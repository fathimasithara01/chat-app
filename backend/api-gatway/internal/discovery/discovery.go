package discovery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/fathima-sithara/api-gateway/internal/config"
	consulapi "github.com/hashicorp/consul/api"
	"go.uber.org/zap"
)

type Discovery interface {
	Lookup(service string) (string, error)
	Close(ctx context.Context) error
}

type staticDiscovery struct {
	m map[string]string
}

func (s *staticDiscovery) Lookup(service string) (string, error) {
	if v, ok := s.m[service]; ok {
		return v, nil
	}
	return "", fmt.Errorf("service not found: %s", service)
}
func (s *staticDiscovery) Close(ctx context.Context) error { return nil }

type consulDiscovery struct {
	client *consulapi.Client
	cache  map[string][]string
	mu     sync.RWMutex
	logger *zap.Logger
}

func (c *consulDiscovery) Lookup(service string) (string, error) {
	c.mu.RLock()
	addrs, ok := c.cache[service]
	c.mu.RUnlock()
	if ok && len(addrs) > 0 {
		// simple: return first
		return addrs[0], nil
	}

	entries, _, err := c.client.Health().Service(service, "", true, nil)
	if err != nil {
		return "", err
	}
	if len(entries) == 0 {
		return "", fmt.Errorf("no healthy instances for %s", service)
	}
	// build addresses
	var urls []string
	for _, e := range entries {
		addr := e.Service.Address
		port := e.Service.Port
		urls = append(urls, fmt.Sprintf("http://%s:%d", addr, port))
	}
	// update cache
	c.mu.Lock()
	c.cache[service] = urls
	c.mu.Unlock()
	return urls[0], nil
}

func (c *consulDiscovery) Close(ctx context.Context) error {
	// no-op for consul client
	_ = c.client
	return nil
}

// NewDiscovery prefers Consul if CONSUL_ADDR provided, otherwise static mapping from SERVICES_JSON
func NewDiscovery(cfg *config.Config, logger *zap.Logger) (Discovery, error) {
	if cfg.ConsulAddr != "" {
		consulCfg := consulapi.DefaultConfig()
		consulCfg.Address = cfg.ConsulAddr
		client, err := consulapi.NewClient(consulCfg)
		if err != nil {
			return nil, err
		}
		d := &consulDiscovery{
			client: client,
			cache:  map[string][]string{},
			logger: logger,
		}
		return d, nil
	}

	if cfg.ServicesJSON == "" {
		return nil, errors.New("SERVICES_JSON or CONSUL_ADDR must be set")
	}
	m := map[string]string{}
	if err := json.Unmarshal([]byte(cfg.ServicesJSON), &m); err != nil {
		return nil, err
	}
	return &staticDiscovery{m: m}, nil
}
