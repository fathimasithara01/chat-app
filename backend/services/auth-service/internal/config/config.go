package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

// Config holds the application's configuration
type Config struct {
	App struct {
		Port int    `yaml:"port"`
		Env  string `yaml:"env"`
		JWT  struct {
			Secret           string `yaml:"secret"`
			AccessTTLMinutes int    `yaml:"accessTTLMinutes"`
			RefreshTTLDays   int    `yaml:"refreshTTLDays"`
		} `yaml:"jwt"`
	} `yaml:"app"`
	Security struct {
		OtpTTLMinutes               int `yaml:"otpTTLMinutes"`
		OtpRateLimitPerPhonePerHour int `yaml:"otpRateLimitPerPhonePerHour"`
	} `yaml:"security"`
	Mongo struct {
		URI      string `yaml:"uri"`
		Database string `yaml:"database"`
	} `yaml:"mongo"`
	Redis struct {
		Addr     string `yaml:"addr"`
		Password string `yaml:"password"`
		DB       int    `yaml:"db"`
	} `yaml:"redis"`
	Twilio struct {
		AccountSID string `yaml:"accountSID"`
		AuthToken  string `yaml:"authToken"`
		From       string `yaml:"from"`
	} `yaml:"twilio"`
	Brevo struct {
		APIKey    string `yaml:"apiKey"`
		FromEmail string `yaml:"fromEmail"`
		FromName  string `yaml:"fromName"`
	} `yaml:"brevo"`
}

// Load reads configuration from a YAML file and environment variables.
func Load(path string) (*Config, error) {
	// Load .env file (if it exists) to get environment variables
	_ = godotenv.Load() // Ignore error if .env doesn't exist

	cfg := &Config{}

	// Read YAML file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config file %s: %w", path, err)
	}

	// Override with environment variables if present
	// This allows sensitive data like API keys to be set via env vars in production
	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		cfg.App.JWT.Secret = secret
	}
	if mongoURI := os.Getenv("MONGO_URI"); mongoURI != "" {
		cfg.Mongo.URI = mongoURI
	}
	if redisAddr := os.Getenv("REDIS_ADDR"); redisAddr != "" {
		cfg.Redis.Addr = redisAddr
	}
	if redisPassword := os.Getenv("REDIS_PASSWORD"); redisPassword != "" {
		cfg.Redis.Password = redisPassword
	}
	if twilioSID := os.Getenv("TWILIO_ACCOUNT_SID"); twilioSID != "" {
		cfg.Twilio.AccountSID = twilioSID
	}
	if twilioAuth := os.Getenv("TWILIO_AUTH_TOKEN"); twilioAuth != "" {
		cfg.Twilio.AuthToken = twilioAuth
	}
	if twilioFrom := os.Getenv("TWILIO_FROM_PHONE"); twilioFrom != "" {
		cfg.Twilio.From = twilioFrom
	}
	if brevoKey := os.Getenv("BREVO_API_KEY"); brevoKey != "" {
		cfg.Brevo.APIKey = brevoKey
	}
	if brevoFromEmail := os.Getenv("BREVO_FROM_EMAIL"); brevoFromEmail != "" {
		cfg.Brevo.FromEmail = brevoFromEmail
	}
	if brevoFromName := os.Getenv("BREVO_FROM_NAME"); brevoFromName != "" {
		cfg.Brevo.FromName = brevoFromName
	}

	return cfg, nil
}
