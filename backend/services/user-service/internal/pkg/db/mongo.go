package db

import (
	"context"
	"log"
	"time"
	"user-service/internal/config"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func ConnectMongo(cfg *config.Config) *mongo.Collection {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoDB.URI))
	if err != nil {
		log.Fatal("MongoDB Connection Failed:", err)
	}

	return client.Database(cfg.MongoDB.Database).Collection(cfg.MongoDB.Collection)
}
