package repository

import (
	"context"
	"time"

	"github.com/fathima-sithara/chat-service/internal/config"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func NewMongoClient() (*mongo.Client, error) {
	// Load config from environment through config.Load()
	cfg, err := LoadEnvDatabaseConfig()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.URI))
	if err != nil {
		return nil, err
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	return client, nil
}

type DatabaseConfig struct {
	URI  string
	Name string
}

/*
This helper reads database config from the main config.
It lets main.go stay clean and avoids passing config everywhere.
*/
func LoadEnvDatabaseConfig() (*DatabaseConfig, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	return &DatabaseConfig{
		URI:  cfg.Database.URI,
		Name: cfg.Database.Name,
	}, nil
}
