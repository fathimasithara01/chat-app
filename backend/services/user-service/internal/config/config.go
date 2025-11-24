package config

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type AppConfig struct {
	Name         string        `yaml:"name"`
	Env          string        `yaml:"env"`
	Port         int           `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout"`
}

type MongoConfig struct {
	URI            string `yaml:"uri"`
	Database       string `yaml:"database"`
	UserCollection string `yaml:"user_collection"`
}

type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type JWTConfig struct {
	Algorithm      string `yaml:"algorithm"`
	PublicKeyPath  string `yaml:"public_key_path"`
	HSSecret       string `yaml:"secret"`
	AccessTTLMin   int    `yaml:"access_ttl_minutes"`
	RefreshTTLDays int    `yaml:"refresh_ttl_days"`
}

type Config struct {
	App   AppConfig   `yaml:"app"`
	Mongo MongoConfig `yaml:"mongo"`
	Redis RedisConfig `yaml:"redis"`
	JWT   JWTConfig   `yaml:"jwt"`
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{}

	data, err := os.ReadFile("config.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to read YAML: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	overrideFromEnv(cfg)

	if err := validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func overrideFromEnv(cfg *Config) {

	if v := os.Getenv("MONGO_URI"); v != "" {
		cfg.Mongo.URI = v
	}
	if v := os.Getenv("MONGO_DB"); v != "" {
		cfg.Mongo.Database = v
	}

	if v := os.Getenv("REDIS_ADDR"); v != "" {
		cfg.Redis.Addr = v
	}
	if v := os.Getenv("REDIS_PASSWORD"); v != "" {
		cfg.Redis.Password = v
	}

	if v := os.Getenv("JWT_SECRET"); v != "" {
		cfg.JWT.HSSecret = v
	}

	if v := os.Getenv("SERVICE_PORT"); v != "" {
		p, _ := strconv.Atoi(v)
		cfg.App.Port = p
	}
}

func validate(cfg *Config) error {
	if cfg.App.Port == 0 {
		return errors.New("app.port is missing or invalid")
	}

	if cfg.Mongo.URI == "" {
		return errors.New("mongo.uri is empty (required MONGO_URI in env)")
	}
	if cfg.Mongo.Database == "" {
		return errors.New("mongo.database is missing")
	}

	if cfg.Redis.Addr == "" {
		return errors.New("redis.addr missing (set REDIS_ADDR)")
	}

	switch cfg.JWT.Algorithm {
	case "RS256":
		if cfg.JWT.PublicKeyPath == "" {
			return errors.New("jwt.public_key_path required for RS256")
		}
	case "HS256":
		if cfg.JWT.HSSecret == "" {
			return errors.New("jwt.secret required for HS256")
		}
	default:
		return errors.New("jwt.algorithm must be RS256 or HS256")
	}

	if cfg.JWT.AccessTTLMin <= 0 || cfg.JWT.RefreshTTLDays <= 0 {
		log.Println("[WARN] Using default JWT TTL values")
	}

	return nil
}
