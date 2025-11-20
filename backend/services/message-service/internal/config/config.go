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
	URI string `yaml:"uri"`
	DB  string `yaml:"db"`
}

type Redis struct {
	Addr string `yaml:"addr"`
	DB   int    `yaml:"db"`
}

type NATS struct {
	URL string `yaml:"url"`
}

type JWT struct {
	Algorithm     string `yaml:"algorithm"`
	PublicKeyPath string `yaml:"public_key_path"`
	HSSecret      string `yaml:"hs_secret"`
}

type Config struct {
	App   App   `yaml:"app"`
	Mongo Mongo `yaml:"mongo"`
	Redis Redis `yaml:"redis"`
	NATS  NATS  `yaml:"nats"`
	JWT   JWT   `yaml:"jwt"`
}

func Load() (*Config, error) {
	cfg := &Config{
		App: App{Port: 8084},
		Mongo: Mongo{
			URI: "mongodb://localhost:27017",
			DB:  "chatapp",
		},
		Redis: Redis{Addr: "localhost:6379", DB: 0},
		NATS:  NATS{URL: "nats://localhost:4222"},
		JWT: JWT{
			Algorithm:     "RS256",
			PublicKeyPath: "./keys/jwt_pub.pem",
			HSSecret:      "",
		},
	}
	if _, err := os.Stat("config.yaml"); err == nil {
		b, _ := os.ReadFile("config.yaml")
		_ = yaml.Unmarshal(b, cfg)
	}
	if cfg.NATS.URL == "" {
		return nil, errors.New("nats.url missing")
	}
	if cfg.JWT.Algorithm == "RS256" && cfg.JWT.PublicKeyPath == "" {
		return nil, errors.New("jwt.public_key_path missing")
	}
	return cfg, nil
}
