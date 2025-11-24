package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type App struct {
	Port int `yaml:"port"`
}

func (a *App) PortString() string { return fmt.Sprintf("%d", a.Port) }

type Mongo struct {
	URI      string `yaml:"uri"`
	Database string `yaml:"database"`
}

type JWTCfg struct {
	PublicKeyPath string `yaml:"public_key_path"`
	Algorithm     string `yaml:"algorithm"`
	Secret        string `yaml:"secret"` 
}

type NATS struct {
	URL string `yaml:"url"`
}

type Kafka struct {
	Brokers []string `yaml:"brokers"`
}

type Config struct {
	App    App    `yaml:"app"`
	Mongo  Mongo  `yaml:"mongo"`
	JWT    JWTCfg `yaml:"jwt"`
	NATS   NATS   `yaml:"nats"`
	Kafka  Kafka  `yaml:"kafka"`
	AESKey string `yaml:"aes_key"`
}

func Load() (*Config, error) {
	cfg := &Config{}

	if _, err := os.Stat("config.yaml"); err == nil {
		b, _ := os.ReadFile("config.yaml")
		if err := yaml.Unmarshal(b, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config.yaml: %w", err)
		}
	}

	_ = godotenv.Load()

	overrideFromEnv(cfg)

	if err := validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func overrideFromEnv(cfg *Config) {
	if v := os.Getenv("PORT"); v != "" {
		fmt.Sscanf(v, "%d", &cfg.App.Port)
	}

	if v := os.Getenv("MONGO_URI"); v != "" {
		cfg.Mongo.URI = v
	}
	if v := os.Getenv("MONGO_DB"); v != "" {
		cfg.Mongo.Database = v
	}

	if v := os.Getenv("JWT_PUBLIC_KEY_PATH"); v != "" {
		cfg.JWT.PublicKeyPath = v
	}
	if v := os.Getenv("JWT_ALGORITHM"); v != "" {
		cfg.JWT.Algorithm = v
	}
	if v := os.Getenv("JWT_SECRET"); v != "" {
		cfg.JWT.Secret = v
	}

	if v := os.Getenv("NATS_URL"); v != "" {
		cfg.NATS.URL = v
	}
	if v := os.Getenv("KAFKA_BROKER"); v != "" {
		cfg.Kafka.Brokers = strings.Split(v, ",")
	}

	if v := os.Getenv("AES256_KEY"); v != "" {
		cfg.AESKey = v
	}
}

func validate(cfg *Config) error {
	if cfg.App.Port == 0 {
		return errors.New("app.port missing or invalid")
	}

	if cfg.Mongo.URI == "" {
		return errors.New("mongo.uri missing")
	}
	if cfg.Mongo.Database == "" {
		return errors.New("mongo.database missing")
	}

	switch strings.ToUpper(cfg.JWT.Algorithm) {
	case "RS256":
		if cfg.JWT.PublicKeyPath == "" {
			return errors.New("jwt.public_key_path required for RS256")
		}
	case "HS256":
		if cfg.JWT.Secret == "" {
			return errors.New("jwt.secret is required for HS256")
		}
	default:
		return errors.New("invalid jwt.algorithm (allowed: RS256, HS256)")
	}

	if cfg.NATS.URL == "" {
		return errors.New("nats.url missing")
	}

	if len(cfg.Kafka.Brokers) == 0 {
		return errors.New("kafka.brokers missing")
	}

	if len(cfg.AESKey) != 32 {
		return errors.New("aes_key must be a 32-byte string for AES-256")
	}

	return nil
}
