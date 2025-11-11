package config

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type App struct {
	Port int `yaml:"port"`
}

func (a *App) PortString() string { return fmt.Sprintf("%d", a.Port) }

type Database struct {
	URI  string `yaml:"uri"`
	Name string `yaml:"name"`
}

type Redis struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type Kafka struct {
	Brokers  []string `yaml:"brokers"`
	TopicIn  string   `yaml:"topic_in"`
	TopicOut string   `yaml:"topic_out"`
	GroupID  string   `yaml:"group_id"`
}

type JWT struct {
	PublicKeyPath string `yaml:"public_key_path"`
}

type Config struct {
	App      App      `yaml:"app"`
	Database Database `yaml:"database"`
	Redis    Redis    `yaml:"redis"`
	Kafka    Kafka    `yaml:"kafka"`
	JWT      JWT      `yaml:"jwt"`
}

func Load() (*Config, error) {
	cfg := &Config{
		App: App{Port: 8083},
		Database: Database{
			URI:  "mongodb://127.0.0.1:27017",
			Name: "chatdb",
		},
		Redis: Redis{Addr: "127.0.0.1:6379", DB: 0},
		Kafka: Kafka{
			Brokers:  []string{"localhost:9092"},
			TopicIn:  "chat_messages_in",
			TopicOut: "chat_messages_out",
			GroupID:  "chat-service-group",
		},
		JWT: JWT{PublicKeyPath: "./keys/jwt_pub.pem"},
	}

	// load config file if present
	if _, err := os.Stat("config/config.yaml"); err == nil {
		b, err := os.ReadFile("config/config.yaml")
		if err != nil {
			return nil, err
		}
		if err := yaml.Unmarshal(b, cfg); err != nil {
			return nil, err
		}
	}

	// environment overrides (optional)
	if v := os.Getenv("MONGODB_URI"); v != "" {
		cfg.Database.URI = v
	}
	if v := os.Getenv("MONGO_NAME"); v != "" {
		cfg.Database.Name = v
	}
	if v := os.Getenv("REDIS_ADDR"); v != "" {
		cfg.Redis.Addr = v
	}
	if v := os.Getenv("KAFKA_BROKER"); v != "" {
		cfg.Kafka.Brokers = []string{v}
	}
	if v := os.Getenv("JWT_PUBLIC_KEY_PATH"); v != "" {
		cfg.JWT.PublicKeyPath = v
	}

	// basic checks
	if cfg.Database.URI == "" {
		return nil, errors.New("database.uri required")
	}
	return cfg, nil
}
