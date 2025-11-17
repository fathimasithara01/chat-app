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
}

type Config struct {
	App   App    `yaml:"app"`
	Mongo Mongo  `yaml:"mongo"`
	JWT   JWTCfg `yaml:"jwt"`
}

func Load() (*Config, error) {
	cfg := &Config{
		App: App{Port: 8083},
		Mongo: Mongo{
			URI:      "mongodb://localhost:27017",
			Database: "chatdb",
		},
		JWT: JWTCfg{
			PublicKeyPath: "./keys/jwt_pub.pem",
		},
	}

	if _, err := os.Stat("config.yaml"); err == nil {
		b, _ := os.ReadFile("config.yaml")
		_ = yaml.Unmarshal(b, cfg)
	}

	if cfg.JWT.PublicKeyPath == "" {
		return nil, errors.New("jwt.public_key_path missing")
	}

	return cfg, nil
}
