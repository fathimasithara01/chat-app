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
	s = strings.Trim(s, `"'`)
	s = strings.TrimPrefix(s, "\uFEFF")
	return s
}

func Load() *Config {
	_ = godotenv.Load()

	cfg := &Config{
		Port:             8086,
		RedisAddr:        "localhost:6379",
		RedisPassword:    "",
		RedisDB:          0,
		PublicKeyPath:    "./keys/jwt_pub.pem",
		RateLimitPerSec:  20,
		EnablePrometheus: false,
	}

	overrideInt := func(key string, dest *int) {
		val := sanitize(os.Getenv(key))
		if val == "" {
			return
		}

		v, err := strconv.Atoi(val)
		if err != nil {
			log.Fatalf("%s must be a number, got: %s", key, val)
		}
		*dest = v
	}

	overrideString := func(key string, dest *string) {
		val := sanitize(os.Getenv(key))
		if val != "" {
			*dest = val
		}
	}

	overrideInt("PORT", &cfg.Port)
	overrideString("REDIS_ADDR", &cfg.RedisAddr)

	if v := sanitize(os.Getenv("REDIS_PASS")); v != "" {
		cfg.RedisPassword = v
	}
	if v := sanitize(os.Getenv("REDIS_PASSWORD")); v != "" {
		cfg.RedisPassword = v
	}

	overrideInt("REDIS_DB", &cfg.RedisDB)
	overrideString("JWT_PUBLIC_KEY_PATH", &cfg.PublicKeyPath)
	overrideInt("RATE_LIMIT_RPS", &cfg.RateLimitPerSec)

	if v := strings.ToLower(sanitize(os.Getenv("PROMETHEUS_ENABLED"))); v != "" {
		cfg.EnablePrometheus = (v == "true" || v == "1" || v == "yes")
	}

	validate(cfg)

	return cfg
}

func validate(cfg *Config) {
	if cfg.Port < 1 || cfg.Port > 65535 {
		log.Fatalf("Invalid PORT: %d (must be 1–65535)", cfg.Port)
	}

	if !strings.Contains(cfg.RedisAddr, ":") {
		log.Fatalf("Invalid REDIS_ADDR format: %s (expected host:port)", cfg.RedisAddr)
	}

	if cfg.PublicKeyPath == "" {
		log.Fatalf("JWT_PUBLIC_KEY_PATH must not be empty")
	}

	if cfg.RateLimitPerSec < 1 {
		log.Fatalf("RATE_LIMIT_RPS must be >= 1")
	}
}

func (c *Config) PortString() string {
	return fmt.Sprintf("%d", c.Port)
}
