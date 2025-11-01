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

type MongoConfig struct {
	URI                    string `mapstructure:"uri"`
	Database               string `mapstructure:"database"`
	MessagesCollection     string `mapstructure:"messages_collection"`
	ConversationsCollection string `mapstructure:"conversations_collection"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type KafkaConfig struct {
	Brokers []string `mapstructure:"brokers"`
	TopicMessageSent string `mapstructure:"topic_message_sent"`
}

type Config struct {
	App   AppConfig   `mapstructure:"app"`
	Mongo MongoConfig `mapstructure:"mongodb"`
	Redis RedisConfig `mapstructure:"redis"`
	Kafka KafkaConfig `mapstructure:"kafka"`
	LogLevel string `mapstructure:"log.level"`
	// derived values
	RequestTimeout time.Duration
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
	// sensible defaults
	c.RequestTimeout = 10 * time.Second
	if c.App.Port == 0 {
		c.App.Port = 8081
	}
	if c.Kafka.TopicMessageSent == "" {
		c.Kafka.TopicMessageSent = "message.sent"
	}
	return &c, nil
}
