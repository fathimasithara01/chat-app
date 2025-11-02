package config

import (
	"os"
	"strconv"
	"time"

	"io/ioutil"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type AppCfg struct {
	Env          string        `yaml:"env"`
	Port         int           `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
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

func Load(path string) (*Config, error) {
	_ = godotenv.Load() // load .env if present

	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := &Config{}
	if err := yaml.Unmarshal(b, cfg); err != nil {
		return nil, err
	}

	// override from env if present
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
	if v := os.Getenv("REDIS_DB"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Redis.DB = n
		}
	}
	if v := os.Getenv("TWILIO_ACCOUNT_SID"); v != "" {
		cfg.Twilio.AccountSID = v
	}
	if v := os.Getenv("TWILIO_AUTH_TOKEN"); v != "" {
		cfg.Twilio.AuthToken = v
	}
	if v := os.Getenv("TWILIO_FROM"); v != "" {
		cfg.Twilio.From = v
	}
	if v := os.Getenv("BREVO_API_KEY"); v != "" {
		cfg.Brevo.APIKey = v
	}
	if v := os.Getenv("BREVO_FROM_EMAIL"); v != "" {
		cfg.Brevo.FromEmail = v
	}
	if v := os.Getenv("JWT_SECRET"); v != "" {
		cfg.App.JWT.Secret = v
	}
	return cfg, nil
}
