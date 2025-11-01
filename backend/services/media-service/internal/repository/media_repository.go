package repository

import (
	"context"
	models "media-service/internal/media"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

type MediaRepo struct {
	col *mongo.Collection
}

func NewMediaRepo(col *mongo.Collection) *MediaRepo {
	return &MediaRepo{col: col}
}

func (r *MediaRepo) Insert(ctx context.Context, m *models.Media) error {
	if m.CreatedAt.IsZero() {
		m.CreatedAt = time.Now().UTC()
	}
	_, err := r.col.InsertOne(ctx, m)
	return err
}

func (r *MediaRepo) GetByID(ctx context.Context, id string) (*models.Media, error) {
	var m models.Media
	err := r.col.FindOne(ctx, map[string]any{"_id": id}).Decode(&m)
	if err != nil {
		return nil, err
	}
	return &m, nil
}
