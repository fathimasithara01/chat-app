package config

import (
	"time"

	"github.com/spf13/viper"
)

type KafkaConfig struct {
	Brokers        []string `mapstructure:"brokers"`
	TopicEvents    string   `mapstructure:"topic_events"`
	GroupID        string   `mapstructure:"group_id"`
	DLQTopic       string   `mapstructure:"dlq_topic"`
	MaxRetries     int      `mapstructure:"max_retries"`
	RetryBackoffMs int      `mapstructure:"retry_backoff_ms"`
}

type EmailConfig struct {
	BrevoAPIKey string `mapstructure:"brevo_api_key"`
	SenderEmail string `mapstructure:"sender_email"`
	SenderName  string `mapstructure:"sender_name"`
}

type SMSConfig struct {
	TwilioSID   string `mapstructure:"twilio_account_sid"`
	TwilioToken string `mapstructure:"twilio_auth_token"`
	FromPhone   string `mapstructure:"from_phone"`
}

type AppConfig struct {
	Env                 string `mapstructure:"env"`
	ShutdownTimeoutSecs int    `mapstructure:"shutdown_timeout_seconds"`
}

type Config struct {
	App   AppConfig   `mapstructure:"app"`
	Kafka KafkaConfig `mapstructure:"kafka"`
	Email EmailConfig `mapstructure:"email"`
	SMS   SMSConfig   `mapstructure:"sms"`
	Log   struct {
		Level string `mapstructure:"level"`
	} `mapstructure:"log"`
	// derived values
	ShutdownTimeout time.Duration
}

func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.AutomaticEnv()
	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	if cfg.App.ShutdownTimeoutSecs == 0 {
		cfg.App.ShutdownTimeoutSecs = 15
	}
	cfg.ShutdownTimeout = time.Duration(cfg.App.ShutdownTimeoutSecs) * time.Second
	if cfg.Kafka.MaxRetries == 0 {
		cfg.Kafka.MaxRetries = 5
	}
	if cfg.Kafka.RetryBackoffMs == 0 {
		cfg.Kafka.RetryBackoffMs = 500
	}
	return &cfg, nil
}
