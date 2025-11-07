package database

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

func ConnectMongo(uri, dbName string, logger *zap.SugaredLogger) (*mongo.Database, *mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		logger.Errorf("MongoDB connection failed: %v", err)
		return nil, nil, err
	}

	if err := client.Ping(ctx, nil); err != nil {
		logger.Errorf("MongoDB ping failed: %v", err)
		return nil, nil, err
	}

	logger.Info("MongoDB connected successfully")
	db := client.Database(dbName)
	return db, client, nil
}
