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

type Config struct {
	App App `yaml:"app"`

	// JWT validation config (RS256 or HS256)
	JWT struct {
		Algorithm     string `yaml:"algorithm"`       // "RS256" or "HS256"
		PublicKeyPath string `yaml:"public_key_path"` // for RS256
		HSSecret      string `yaml:"hs_secret"`       // for HS256
	} `yaml:"jwt"`
}

func Load() (*Config, error) {
	cfg := &Config{}
	// defaults
	cfg.App.Port = 8083
	cfg.JWT.Algorithm = "RS256"
	cfg.JWT.PublicKeyPath = "./keys/jwt_pub.pem"
	cfg.JWT.HSSecret = ""

	// optional yaml
	if _, err := os.Stat("config.yaml"); err == nil {
		b, _ := os.ReadFile("config.yaml")
		_ = yaml.Unmarshal(b, cfg)
	}

	// validate minimal
	if cfg.JWT.Algorithm == "RS256" && cfg.JWT.PublicKeyPath == "" {
		return nil, errors.New("jwt.public_key_path required for RS256")
	}
	if cfg.JWT.Algorithm == "HS256" && cfg.JWT.HSSecret == "" {
		return nil, errors.New("jwt.hs_secret required for HS256")
	}
	return cfg, nil
}
