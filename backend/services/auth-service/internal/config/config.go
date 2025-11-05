package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv" // Ensure this is imported for .env loading
	"gopkg.in/yaml.v3"
)

type AppCfg struct {
	Env          string        `yaml:"env"`
	Port         int           `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout"` // Added IdleTimeout
	JWT          struct {
		Secret           string `yaml:"secret"`
		AccessTTLMinutes int    `yaml:"accessTTLMinutes"`
		RefreshTTLDays   int    `yaml:"refreshTTLDays"`
	} `yaml:"jwt"`
}

type MongoCfg struct {
	URI      string `yaml:"uri"`
	Database string `yaml:"database"`
}

type RedisCfg struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type TwilioCfg struct {
	AccountSID string `yaml:"accountSID"`
	AuthToken  string `yaml:"authToken"`
	From       string `yaml:"from"`
}

type BrevoCfg struct {
	APIKey    string `yaml:"apiKey"`
	FromEmail string `yaml:"fromEmail"`
	FromName  string `yaml:"fromName"`
}

type UserCfg struct {
	Collection string `yaml:"collection"`
}

type SecurityCfg struct {
	OtpTTLMinutes               int `yaml:"otpTTLMinutes"`
	OtpRateLimitPerPhonePerHour int `yaml:"otpRateLimitPerPhonePerHour"`
	PasswordHashCost            int `yaml:"passwordHashCost"` // Added bcrypt cost
}

type Config struct {
	App      AppCfg      `yaml:"app"`
	Mongo    MongoCfg    `yaml:"mongo"`
	Redis    RedisCfg    `yaml:"redis"`
	Twilio   TwilioCfg   `yaml:"twilio"`
	Brevo    BrevoCfg    `yaml:"brevo"`
	User     UserCfg     `yaml:"user"`
	Security SecurityCfg `yaml:"security"`
}

// Load reads the configuration from a YAML file and overrides with environment variables.
func Load(path string) (*Config, error) {
	// Load .env variables first, but allow main.go to handle errors if the file is missing
	_ = godotenv.Load()

	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(b, cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config YAML: %w", err)
	}

	// Override with environment variables (if set)
	// App Configuration
	if v := os.Getenv("APP_ENV"); v != "" {
		cfg.App.Env = v
	}
	if v := os.Getenv("APP_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.App.Port = n
		}
	}
	if v := os.Getenv("JWT_SECRET"); v != "" {
		cfg.App.JWT.Secret = v
	}
	if v := os.Getenv("JWT_ACCESS_TTL_MINUTES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.App.JWT.AccessTTLMinutes = n
		}
	}
	if v := os.Getenv("JWT_REFRESH_TTL_DAYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.App.JWT.RefreshTTLDays = n
		}
	}

	// Mongo Configuration
	if v := os.Getenv("MONGO_URI"); v != "" {
		cfg.Mongo.URI = v
	}
	if v := os.Getenv("MONGO_DB"); v != "" {
		cfg.Mongo.Database = v
	}

	// Redis Configuration
	if v := os.Getenv("REDIS_ADDR"); v != "" {
		cfg.Redis.Addr = v
	}
	if v := os.Getenv("REDIS_PASSWORD"); v != "" {
		cfg.Redis.Password = v
	}
	if v := os.Getenv("REDIS_DB"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Redis.DB = n
		}
	}

	// Twilio Configuration
	if v := os.Getenv("TWILIO_ACCOUNT_SID"); v != "" {
		cfg.Twilio.AccountSID = v
	}
	if v := os.Getenv("TWILIO_AUTH_TOKEN"); v != "" {
		cfg.Twilio.AuthToken = v
	}
	if v := os.Getenv("TWILIO_FROM"); v != "" {
		cfg.Twilio.From = v
	}

	// Brevo Configuration
	if v := os.Getenv("BREVO_API_KEY"); v != "" {
		cfg.Brevo.APIKey = v
	}
	if v := os.Getenv("BREVO_FROM_EMAIL"); v != "" {
		cfg.Brevo.FromEmail = v
	}
	if v := os.Getenv("BREVO_FROM_NAME"); v != "" {
		cfg.Brevo.FromName = v
	}

	// Security Configuration
	if v := os.Getenv("OTP_TTL_MINUTES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Security.OtpTTLMinutes = n
		}
	}
	if v := os.Getenv("OTP_RATE_LIMIT_PER_PHONE_PER_HOUR"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Security.OtpRateLimitPerPhonePerHour = n
		}
	}
	if v := os.Getenv("PASSWORD_HASH_COST"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Security.PasswordHashCost = n
		}
	}

	// Basic validation (can be extended)
	if cfg.App.JWT.Secret == "" {
		return nil, errors.New("JWT secret is not configured. Set JWT_SECRET in .env or config.yaml")
	}

	return cfg, nil
}