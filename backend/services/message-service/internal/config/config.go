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

type Mongo struct {
	URI      string `yaml:"uri"`
	Database string `yaml:"database"`
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

type Media struct {
	Provider          string `yaml:"provider"`
	Bucket            string `yaml:"bucket"`
	PublicBaseURL     string `yaml:"public_base_url"`
	PresignExpirySecs int    `yaml:"presign_expiry_seconds"`
}

type Security struct {
	AES256Key string `yaml:"aes256_key"`
}

type Config struct {
	App      App      `yaml:"app"`
	Mongo    Mongo    `yaml:"mongo"`
	Redis    Redis    `yaml:"redis"`
	Kafka    Kafka    `yaml:"kafka"`
	Media    Media    `yaml:"media"`
	Security Security `yaml:"security"`
}

func Load() (*Config, error) {
	cfg := &Config{
		App: App{Port: 8085},
		Mongo: Mongo{
			URI:      "mongodb://mongo:27017",
			Database: "chat_message_db",
		},
		Redis: Redis{Addr: "redis:6379", DB: 0},
		Kafka: Kafka{
			Brokers:  []string{"kafka:9092"},
			TopicIn:  "chat_messages_in",
			TopicOut: "chat_messages_out",
			GroupID:  "message-service-group",
		},
		Media: Media{
			Provider:          "minio",
			Bucket:            "chat-media",
			PublicBaseURL:     "http://localhost:9000/chat-media",
			PresignExpirySecs: 7200,
		},
	}

	if _, err := os.Stat("config.yaml"); err == nil {
		b, _ := os.ReadFile("config.yaml")
		_ = yaml.Unmarshal(b, cfg)
	}

	if v := os.Getenv("AES256_KEY"); v != "" {
		cfg.Security.AES256Key = v
	}
	if cfg.Security.AES256Key == "" {
		return nil, errors.New("AES256_KEY must be set (32 bytes)")
	}
	return cfg, nil
}
