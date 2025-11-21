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

// sanitize removes whitespace, quotes & BOM issues.
func sanitize(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, `"`)
	s = strings.Trim(s, `'`)

	// Remove BOM if present
	s = strings.TrimPrefix(s, "\uFEFF")

	return s
}

func Load() *Config {
	// Load .env silently, even if missing
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

	// override values
	overrideInt("PORT", &cfg.Port)
	overrideString("REDIS_ADDR", &cfg.RedisAddr)
	overrideString("REDIS_PASS", &cfg.RedisPassword)
	overrideInt("REDIS_DB", &cfg.RedisDB)

	overrideString("JWT_PUBLIC_KEY_PATH", &cfg.PublicKeyPath)
	overrideInt("RATE_LIMIT_RPS", &cfg.RateLimitPerSec)

	if raw := sanitize(os.Getenv("PROMETHEUS_ENABLED")); raw == "true" {
		cfg.EnablePrometheus = true
	}

	// Validation: Redis address must include ":"
	if !strings.Contains(cfg.RedisAddr, ":") {
		log.Fatalf("❌ Invalid REDIS_ADDR: %s (must be host:port)", cfg.RedisAddr)
	}

	if cfg.Port <= 0 || cfg.Port > 65535 {
		log.Fatalf("❌ Invalid PORT: %d", cfg.Port)
	}

	// Pretty print configuration
	fmt.Println("────────────────────────────────────────────────────────")
	fmt.Println("              🚀 CONFIGURATION LOADED")
	fmt.Println("────────────────────────────────────────────────────────")
	fmt.Printf("  ENV Loaded:          %v\n", cfg.EnvLoaded)
	fmt.Printf("  Service Port:        %d\n", cfg.Port)
	fmt.Printf("  Redis Address:       %s\n", cfg.RedisAddr)
	fmt.Printf("  Redis DB:            %d\n", cfg.RedisDB)
	fmt.Printf("  JWT Public Key:      %s\n", cfg.PublicKeyPath)
	fmt.Printf("  Rate Limit (RPS):    %d\n", cfg.RateLimitPerSec)
	fmt.Printf("  Prometheus Enabled:  %v\n", cfg.EnablePrometheus)
	fmt.Println("────────────────────────────────────────────────────────")

	return cfg
}

func (c *Config) PortString() string {
	return fmt.Sprintf("%d", c.Port)
}
