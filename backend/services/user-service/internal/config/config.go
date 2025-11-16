package config

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type Config struct {
	App struct {
		Name         string        `yaml:"name"`
		Env          string        `yaml:"env"`
		Port         int           `yaml:"port"`
		ReadTimeout  time.Duration `yaml:"read_timeout"`
		WriteTimeout time.Duration `yaml:"write_timeout"`
		IdleTimeout  time.Duration `yaml:"idle_timeout"`
	} `yaml:"app"`

	Mongo struct {
		Database       string `yaml:"database"`
		UserCollection string `yaml:"user_collection"`
	} `yaml:"mongo"`

	Redis struct {
		DB int `yaml:"db"`
	} `yaml:"redis"`

	JWT struct {
		AccessTTLMinutes int `yaml:"access_ttl_minutes"`
		RefreshTTLDays   int `yaml:"refresh_ttl_days"`
	} `yaml:"jwt"`

	MongoURI  string
	RedisAddr string
	RedisPass string
	JWTSecret string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{}

	data, err := os.ReadFile("config/config.yaml")
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	cfg.MongoURI = os.Getenv("MONGO_URI")
	cfg.RedisAddr = os.Getenv("REDIS_ADDR")
	cfg.RedisPass = os.Getenv("REDIS_PASSWORD")
	cfg.JWTSecret = os.Getenv("JWT_SECRET")

	if cfg.MongoURI == "" || cfg.JWTSecret == "" {
		log.Fatal(" Required environment variables missing in .env")
	}

	return cfg, nil
}
