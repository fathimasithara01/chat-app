package config

import (
	"github.com/spf13/viper"
)

type JWTConfig struct {
	Secret           string `mapstructure:"secret"`
	AccessTTLMinutes int    `mapstructure:"access_ttl_minutes"`
	RefreshTTLDays   int    `mapstructure:"refresh_ttl_days"`
}

type AppConfig struct {
	Env  string    `mapstructure:"env"`
	Port int       `mapstructure:"port"`
	JWT  JWTConfig `mapstructure:"jwt"`
}

type MongoConfig struct {
	URI      string `mapstructure:"uri"`
	Database string `mapstructure:"database"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type TwilioConfig struct {
	AccountSID string `mapstructure:"account_sid"`
	AuthToken  string `mapstructure:"auth_token"`
	From       string `mapstructure:"from"`
}

type BrevoConfig struct {
	APIKey    string `mapstructure:"api_key"`
	FromEmail string `mapstructure:"from_email"`
	FromName  string `mapstructure:"from_name"`
}

type SecurityConfig struct {
	OtpTTLMinutes               int `mapstructure:"otp_ttl_minutes"`
	OtpRateLimitPerPhonePerHour int `mapstructure:"otp_rate_limit_per_phone_per_hour"`
}

type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Mongo    MongoConfig    `mapstructure:"mongo"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Twilio   TwilioConfig   `mapstructure:"twilio"`
	Brevo    BrevoConfig    `mapstructure:"brevo"`
	Security SecurityConfig `mapstructure:"security"`
	LogLevel string         `mapstructure:"log.level"`
}

func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.AutomaticEnv()
	v.SetEnvPrefix("AUTH")

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	var c Config
	if err := v.Unmarshal(&c); err != nil {
		return nil, err
	}

	if c.App.JWT.AccessTTLMinutes == 0 {
		c.App.JWT.AccessTTLMinutes = 15
	}

	if c.App.JWT.RefreshTTLDays == 0 {
		c.App.JWT.RefreshTTLDays = 30
	}

	if c.Security.OtpTTLMinutes == 0 {
		c.Security.OtpTTLMinutes = 5
	}

	return &c, nil
}
