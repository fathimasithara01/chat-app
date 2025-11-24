package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type AppConfig struct {
	Port int `yaml:"port"`
}

func (a *AppConfig) PortString() string {
	return fmt.Sprintf("%d", a.Port)
}

type JWTConfig struct {
	Algorithm     string `yaml:"algorithm"`
	PublicKeyPath string `yaml:"public_key_path"`
	HSSecret      string `yaml:"hs_secret"`
}

type Config struct {
	App AppConfig `yaml:"app"`
	JWT JWTConfig `yaml:"jwt"`

	EnvLoaded bool
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{}

	data, err := os.ReadFile("config.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to read config.yaml: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	if os.Getenv("PORT") != "" {
		cfg.EnvLoaded = true
	}

	overrideIntEnv(&cfg.App.Port, "PORT")
	overrideStringEnv(&cfg.JWT.Algorithm, "JWT_ALG")
	overrideStringEnv(&cfg.JWT.PublicKeyPath, "JWT_PUBLIC_KEY_PATH")
	overrideStringEnv(&cfg.JWT.HSSecret, "JWT_SECRET")

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func overrideStringEnv(dest *string, key string) {
	if val := os.Getenv(key); val != "" {
		*dest = sanitize(val)
	}
}

func overrideIntEnv(dest *int, key string) {
	if val := os.Getenv(key); val != "" {
		fmt.Sscanf(val, "%d", dest)
	}
}

func sanitize(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, `"`)
	return s
}

func (c *Config) validate() error {
	if c.App.Port <= 0 || c.App.Port > 65535 {
		return errors.New("app.port is invalid (must be 1-65535)")
	}

	algo := strings.ToUpper(c.JWT.Algorithm)
	switch algo {
	case "RS256":
		if c.JWT.PublicKeyPath == "" {
			return errors.New("jwt.public_key_path is required for RS256")
		}
		if _, err := os.Stat(c.JWT.PublicKeyPath); err != nil {
			return fmt.Errorf("rs256 public key not found: %s", c.JWT.PublicKeyPath)
		}

	case "HS256":
		if c.JWT.HSSecret == "" {
			return errors.New("jwt.hs_secret is required for HS256")
		}

	default:
		return fmt.Errorf("unsupported jwt.algorithm: %s (use RS256 or HS256)", algo)
	}

	return nil
}
