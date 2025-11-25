package config

import (
	"log"
	"os"
)

type Config struct {
	Port         string
	MongoURI     string
	MongoDB      string
	KafkaBrokers string
	KafkaTopic   string
}

func Load() *Config {
	return &Config{
		Port:         getEnv("PORT", "8087"),
		MongoURI:     getEnv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDB:      getEnv("MONGO_DB", "chatapp"),
		KafkaBrokers: getEnv("KAFKA_BROKERS", "localhost:9092"),
		KafkaTopic:   getEnv("KAFKA_TOPIC", "notifications"),
	}
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	log.Println("Using default:", key, defaultVal)
	return defaultVal
}
