package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type App struct {
	Env     string `yaml:"env"`
	Port    int    `yaml:"port"`
	Timeout string `yaml:"shutdown_timeout"`
	Rate    int    `yaml:"rate_limit_per_min"`
}

func (a *App) PortString() string { return fmt.Sprintf("%d", a.Port) }

type Mongo struct {
	URI string `yaml:"uri"`
	DB  string `yaml:"db"`
}

type Redis struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type NATS struct {
	URL string `yaml:"url"`
}

type Kafka struct {
	Brokers  []string `yaml:"brokers"`
	TopicIn  string   `yaml:"topic_in"`
	TopicOut string   `yaml:"topic_out"`
}

type JWT struct {
	Alg           string `yaml:"alg"`
	PublicKeyPath string `yaml:"public_key_path"`
	HSSecret      string `yaml:"hs_secret"`
}

type Config struct {
	App   App   `yaml:"app"`
	Mongo Mongo `yaml:"mongo"`
	Redis Redis `yaml:"redis"`
	NATS  NATS  `yaml:"nats"`
	Kafka Kafka `yaml:"kafka"`
	JWT   JWT   `yaml:"jwt"`
}

func Load() (*Config, error) {
	cfg := &Config{}

	if _, err := os.Stat("config.yaml"); err == nil {
		b, _ := os.ReadFile("config.yaml")
		if err := yaml.Unmarshal(b, cfg); err != nil {
			return nil, err
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
	if v := os.Getenv("APP_ENV"); v != "" {
		cfg.App.Env = v
	}
	if v := os.Getenv("SERVICE_PORT"); v != "" {
		n, _ := strconv.Atoi(v)
		cfg.App.Port = n
	}

	if v := os.Getenv("MONGODB_URI"); v != "" {
		cfg.Mongo.URI = v
	}
	if v := os.Getenv("MONGO_NAME"); v != "" {
		cfg.Mongo.DB = v
	}

	if v := os.Getenv("REDIS_ADDR"); v != "" {
		cfg.Redis.Addr = v
	}
	if v := os.Getenv("REDIS_PASSWORD"); v != "" {
		cfg.Redis.Password = v
	}

	if v := os.Getenv("KAFKA_BROKER"); v != "" {
		cfg.Kafka.Brokers = strings.Split(v, ",")
	}


	if v := os.Getenv("JWT_PUBLIC_KEY_PATH"); v != "" {
		cfg.JWT.PublicKeyPath = v
	}
	if v := os.Getenv("JWT_SECRET"); v != "" {
		cfg.JWT.HSSecret = v
	}

}

func validate(cfg *Config) error {
	if cfg.App.Port == 0 {
		return errors.New("app.port missing or invalid")
	}

	if cfg.Mongo.URI == "" {
		return errors.New("mongo.uri missing")
	}
	if cfg.Mongo.DB == "" {
		return errors.New("mongo.db missing")
	}

	if cfg.Redis.Addr == "" {
		return errors.New("redis.addr missing")
	}

	if cfg.NATS.URL == "" {
		return errors.New("nats.url missing")
	}

	if len(cfg.Kafka.Brokers) == 0 {
		return errors.New("kafka.brokers missing")
	}
	if cfg.Kafka.TopicIn == "" || cfg.Kafka.TopicOut == "" {
		return errors.New("kafka topics missing")
	}

	switch strings.ToUpper(cfg.JWT.Alg) {
	case "RS256":
		if cfg.JWT.PublicKeyPath == "" {
			return errors.New("jwt.public_key_path required for RS256")
		}
	case "HS256":
		if cfg.JWT.HSSecret == "" {
			return errors.New("jwt.hs_secret required for HS256")
		}
	default:
		return errors.New("invalid jwt.alg (use RS256 or HS256)")
	}

	return nil
}
