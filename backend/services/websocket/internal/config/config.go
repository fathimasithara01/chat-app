package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port             int
	RedisAddr        string
	RedisPassword    string
	RedisDB          int
	PublicKeyPath    string
	RateLimitPerSec  int
	EnablePrometheus bool
	EnvLoaded        bool
}

func sanitize(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, `"`)
	s = strings.Trim(s, `'`)
	s = strings.TrimPrefix(s, "\uFEFF")
	return s
}

func Load() *Config {
	err := godotenv.Load()
	envLoaded := err == nil

	cfg := &Config{
		Port:             8086,
		RedisAddr:        "localhost:6379",
		RedisPassword:    "",
		RedisDB:          0,
		PublicKeyPath:    "./keys/jwt_pub.pem",
		RateLimitPerSec:  20,
		EnablePrometheus: false,
		EnvLoaded:        envLoaded,
	}

	overrideInt := func(key string, dest *int) {
		if raw := os.Getenv(key); raw != "" {
			raw = sanitize(raw)
			if v, err := strconv.Atoi(raw); err == nil {
				*dest = v
			} else {
				log.Fatalf("❌ Invalid value for %s: %s", key, raw)
			}
		}
	}

	overrideString := func(key string, dest *string) {
		if raw := os.Getenv(key); raw != "" {
			*dest = sanitize(raw)
		}
	}

	overrideInt("PORT", &cfg.Port)
	overrideString("REDIS_ADDR", &cfg.RedisAddr)
	if raw := os.Getenv("REDIS_PASS"); raw != "" {
		cfg.RedisPassword = sanitize(raw)
	} else {
		overrideString("REDIS_PASSWORD", &cfg.RedisPassword)
	}
	overrideInt("REDIS_DB", &cfg.RedisDB)

	overrideString("JWT_PUBLIC_KEY_PATH", &cfg.PublicKeyPath)
	overrideInt("RATE_LIMIT_RPS", &cfg.RateLimitPerSec)

	if raw := sanitize(os.Getenv("PROMETHEUS_ENABLED")); raw == "true" {
		cfg.EnablePrometheus = true
	}

	if !strings.Contains(cfg.RedisAddr, ":") {
		log.Fatalf(" Invalid REDIS_ADDR: %s (must be host:port)", cfg.RedisAddr)
	}

	if cfg.Port <= 0 || cfg.Port > 65535 {
		log.Fatalf(" Invalid PORT: %d", cfg.Port)
	}

	return cfg
}

func (c *Config) PortString() string {
	return fmt.Sprintf("%d", c.Port)
}
