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

type JWTCfg struct {
	PublicKeyPath string `yaml:"public_key_path"`
	Algorithm     string `yaml:"algorithm"` // "RS256" or "HS256"
	Secret        string `yaml:"secret"`    // for HS256
}

type NATS struct {
	URL string `yaml:"url"`
}

type Config struct {
	App   App    `yaml:"app"`
	Mongo Mongo  `yaml:"mongo"`
	JWT   JWTCfg `yaml:"jwt"`
	NATS  NATS   `yaml:"nats"`
}

func Load() (*Config, error) {
	cfg := &Config{
		App: App{Port: 8083},
		Mongo: Mongo{
			URI:      "mongodb://localhost:27017",
			Database: "chatapp",
		},
		JWT: JWTCfg{
			PublicKeyPath: "./keys/jwt_pub.pem",
			Algorithm:     "RS256",
			Secret:        "",
		},
		NATS: NATS{URL: "nats://localhost:4222"},
	}

	if _, err := os.Stat("config.yaml"); err == nil {
		b, _ := os.ReadFile("config.yaml")
		_ = yaml.Unmarshal(b, cfg)
	}

	if cfg.JWT.PublicKeyPath == "" && cfg.JWT.Secret == "" && cfg.JWT.Algorithm == "RS256" {
		return nil, errors.New("jwt.public_key_path missing for RS256")
	}
	if cfg.NATS.URL == "" {
		return nil, errors.New("nats.url missing")
	}

	return cfg, nil
}
