package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type AppCfg struct {
	Env          string        `yaml:"env"`
	Port         int           `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout"`
	JWT          struct {
		// Secret           string `yaml:"secret"`
		PrivateKeyPath   string `yaml:"privateKeyPath"`
		PublicKeyPath    string `yaml:"publicKeyPath"`
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

type EmailJSCfg struct {
	ServiceID   string `yaml:"serviceID"`
	TemplateID  string `yaml:"templateID"`
	PublicKey   string `yaml:"publicKey"`
	PrivateKey  string `yaml:"privateKey"`
	SenderEmail string `yaml:"senderEmail"`
	Enabled     bool   `yaml:"enabled"`
}

type UserCfg struct {
	Collection string `yaml:"collection"`
}

type SecurityCfg struct {
	OtpTTLMinutes               int `yaml:"otpTTLMinutes"`
	OtpRateLimitPerPhonePerHour int `yaml:"otpRateLimitPerPhonePerHour"`
	PasswordHashCost            int `yaml:"passwordHashCost"`
}

type Config struct {
	App      AppCfg      `yaml:"app"`
	Mongo    MongoCfg    `yaml:"mongo"`
	Redis    RedisCfg    `yaml:"redis"`
	Twilio   TwilioCfg   `yaml:"twilio"`
	EmailJS  EmailJSCfg  `yaml:"emailjs"`
	User     UserCfg     `yaml:"user"`
	Security SecurityCfg `yaml:"security"`
}

func Load(path string) (*Config, error) {
	_ = godotenv.Load()

	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	cfg := &Config{}
	if err := yaml.Unmarshal(b, cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config YAML: %w", err)
	}

	override := func(env string, apply func(string)) {
		if v := os.Getenv(env); v != "" {
			apply(v)
		}
	}

	override("APP_ENV", func(v string) { cfg.App.Env = v })
	override("APP_PORT", func(v string) {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.App.Port = n
		}
	})
	// override("JWT_SECRET", func(v string) { cfg.App.JWT.Secret = v })
	override("JWT_PRIVATE_KEY_PATH", func(v string) { cfg.App.JWT.PrivateKeyPath = v })
	override("JWT_PUBLIC_KEY_PATH", func(v string) { cfg.App.JWT.PublicKeyPath = v })
	override("MONGO_URI", func(v string) { cfg.Mongo.URI = v })
	override("MONGO_DB", func(v string) { cfg.Mongo.Database = v })
	override("REDIS_ADDR", func(v string) { cfg.Redis.Addr = v })
	override("REDIS_PASSWORD", func(v string) { cfg.Redis.Password = v })
	override("EMAILJS_SERVICE_ID", func(v string) { cfg.EmailJS.ServiceID = v })
	override("EMAILJS_TEMPLATE_ID", func(v string) { cfg.EmailJS.TemplateID = v })
	override("EMAILJS_PUBLIC_KEY", func(v string) { cfg.EmailJS.PublicKey = v })
	override("EMAILJS_PRIVATE_KEY", func(v string) { cfg.EmailJS.PrivateKey = v })
	override("EMAILJS_SENDER_EMAIL", func(v string) { cfg.EmailJS.SenderEmail = v })

	if v := os.Getenv("EMAILJS_ENABLED"); v == "true" {
		cfg.EmailJS.Enabled = true
	}

	override("JWT_ACCESS_TTL_MINUTES", func(v string) {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.App.JWT.AccessTTLMinutes = n
		}
	})
	override("JWT_REFRESH_TTL_DAYS", func(v string) {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.App.JWT.RefreshTTLDays = n
		}
	})
	override("OTP_TTL_MINUTES", func(v string) {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Security.OtpTTLMinutes = n
		}
	})
	override("OTP_RATE_LIMIT_PER_PHONE_PER_HOUR", func(v string) {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Security.OtpRateLimitPerPhonePerHour = n
		}
	})
	override("PASSWORD_HASH_COST", func(v string) {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Security.PasswordHashCost = n
		}
	})

	// if cfg.App.JWT.Secret == "" {
	// 	return nil, errors.New("JWT_SECRET is required (set in .env or config.yaml)")
	// }
	if cfg.App.JWT.PrivateKeyPath == "" || cfg.App.JWT.PublicKeyPath == "" {
		return nil, errors.New("JWT_PRIVATE_KEY_PATH and JWT_PUBLIC_KEY_PATH are required")
	}

	if cfg.Mongo.URI == "" {
		return nil, errors.New("MONGO_URI is required")
	}
	if cfg.EmailJS.Enabled && (cfg.EmailJS.ServiceID == "" || cfg.EmailJS.TemplateID == "" || cfg.EmailJS.PublicKey == "") {
		return nil, errors.New("EmailJS enabled but missing ServiceID, TemplateID, or PublicKey")
	}

	return cfg, nil
}
