package config

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	AppEnv          string
	AppPort         string
	ShutdownTimeout time.Duration
	RateLimitPerMin int

	MongoURI string
	MongoDB  string

	RedisAddr     string
	RedisPassword string
	RedisDB       int

	KafkaBrokers  []string
	KafkaTopicIn  string
	KafkaTopicOut string

	JWTPublicKeyPath string
	JWTAlg           string
}

func Load() *Config {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("failed to get working directory: %v", err)
	}

	// Absolute path to config folder
	configPath := filepath.Join(wd, "config")

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configPath)

	log.Println("Looking for config.yaml at:", configPath)

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("error reading config.yaml: %v, tried path: %s", err, viper.ConfigFileUsed())
	}

	return &Config{
		AppEnv:          viper.GetString("app_env"),
		AppPort:         viper.GetString("app_port"),
		ShutdownTimeout: viper.GetDuration("shutdown_timeout"),
		RateLimitPerMin: viper.GetInt("rate_limit_per_min"),

		MongoURI: viper.GetString("mongo_uri"),
		MongoDB:  viper.GetString("mongo_db"),

		RedisAddr:     viper.GetString("redis_addr"),
		RedisPassword: viper.GetString("redis_password"),
		RedisDB:       viper.GetInt("redis_db"),

		KafkaBrokers:  viper.GetStringSlice("kafka_brokers"),
		KafkaTopicIn:  viper.GetString("kafka_topic_in"),
		KafkaTopicOut: viper.GetString("kafka_topic_out"),

		JWTPublicKeyPath: viper.GetString("jwt_public_key_path"),
		JWTAlg:           viper.GetString("jwt_alg"),
	}
}
