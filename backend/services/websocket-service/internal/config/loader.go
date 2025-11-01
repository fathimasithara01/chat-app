package config

import (
	"time"

	"github.com/spf13/viper"
)

type AppConfig struct {
	Env       string `mapstructure:"env"`
	Port      int    `mapstructure:"port"`
	JWTSecret string `mapstructure:"jwt_secret"`
}

type RedisConfig struct {
	Addr   string `mapstructure:"addr"`
	Pass   string `mapstructure:"password"`
	DB     int    `mapstructure:"db"`
	Prefix string `mapstructure:"prefix"`
}

type KafkaConfig struct {
	Brokers           []string `mapstructure:"brokers"`
	TopicMessageSent  string   `mapstructure:"topic_message_sent"`
}

type WSConfig struct {
	PingIntervalSeconds      int `mapstructure:"ping_interval_seconds"`
	WriteDeadlineSeconds     int `mapstructure:"write_deadline_seconds"`
	MaxMessageSizeBytes      int64 `mapstructure:"max_message_size_bytes"`
}

type Config struct {
	App   AppConfig `mapstructure:"app"`
	Redis RedisConfig `mapstructure:"redis"`
	Kafka KafkaConfig `mapstructure:"kafka"`
	WS    WSConfig `mapstructure:"ws"`
	LogLevel string `mapstructure:"log.level"`

	// derived/timeouts
	PingInterval  time.Duration
	WriteDeadline time.Duration
}

func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}
	var c Config
	if err := v.Unmarshal(&c); err != nil {
		return nil, err
	}
	if c.WS.PingIntervalSeconds == 0 {
		c.WS.PingIntervalSeconds = 25
	}
	if c.WS.WriteDeadlineSeconds == 0 {
		c.WS.WriteDeadlineSeconds = 10
	}
	c.PingInterval = time.Duration(c.WS.PingIntervalSeconds) * time.Second
	c.WriteDeadline = time.Duration(c.WS.WriteDeadlineSeconds) * time.Second
	if c.WS.MaxMessageSizeBytes == 0 {
		c.WS.MaxMessageSizeBytes = 65536
	}
	return &c, nil
}
