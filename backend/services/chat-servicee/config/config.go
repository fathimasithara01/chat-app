package config

import (
	"fmt"
	"log"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration values
type Config struct {
	AppEnv          string
	AppPort         string
	ShutdownTimeout time.Duration

	// MongoDB
	MongoURI string
	MongoDB  string

	// Redis
	RedisAddr string
	RedisPwd  string
	RedisDB   int

	// Kafka
	KafkaBrokers  []string
	KafkaTopicIn  string
	KafkaTopicOut string
	KafkaGroupID  string

	// JWT
	JWTPublicKeyPath  string
	JWTPrivateKeyPath string
	JWTAlg            string

	// Rate limiting
	RateLimitPerMin int
}

// Load reads configuration from config.yaml or environment variables
func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	viper.AddConfigPath(".")

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config.yaml: %w", err)
	}

	cfg := &Config{
		AppEnv:            viper.GetString("APP_ENV"),
		AppPort:           viper.GetString("APP_PORT"),
		ShutdownTimeout:   viper.GetDuration("SHUTDOWN_TIMEOUT") * time.Second,
		MongoURI:          viper.GetString("MONGO_URI"),
		MongoDB:           viper.GetString("MONGO_DB"),
		RedisAddr:         viper.GetString("REDIS_ADDR"),
		RedisPwd:          viper.GetString("REDIS_PASSWORD"),
		RedisDB:           viper.GetInt("REDIS_DB"),
		KafkaBrokers:      viper.GetStringSlice("KAFKA_BROKERS"),
		KafkaTopicIn:      viper.GetString("KAFKA_TOPIC_IN"),
		KafkaTopicOut:     viper.GetString("KAFKA_TOPIC_OUT"),
		KafkaGroupID:      viper.GetString("KAFKA_GROUP_ID"),
		JWTPublicKeyPath:  viper.GetString("JWT_PUBLIC_KEY_PATH"),
		JWTPrivateKeyPath: viper.GetString("JWT_PRIVATE_KEY_PATH"),
		JWTAlg:            viper.GetString("JWT_ALG"),
		RateLimitPerMin:   viper.GetInt("RATE_LIMIT_PER_MIN"),
	}

	log.Printf("Looking for config.yaml at: %s\n", viper.ConfigFileUsed())
	return cfg, nil
}
