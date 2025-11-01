package service

import (
	"bytes"
	"context"
	"image"
	models "media-service/internal/media"
	"media-service/internal/repository"
	"media-service/internal/storage"
	utils "media-service/internal/utis"
	"time"

	"github.com/disintegration/imaging"
)

type MediaService struct {
	repo       *repository.MediaRepo
	store      *storage.S3Store
	cache      Cache // interface for caching signed url if used (optional)
	presignTTL time.Duration
}

type Cache interface {
	Set(ctx context.Context, key string, val string, ttl time.Duration) error
	Get(ctx context.Context, key string) (string, error)
}

func NewMediaService(repo *repository.MediaRepo, store *storage.S3Store, presignTTL time.Duration) *MediaService {
	return &MediaService{repo: repo, store: store, presignTTL: presignTTL}
}
func (s *MediaService) UploadFile(ctx context.Context, userID, filename, contentType string, data []byte) (*models.Media, error) {
	id := utils.NewID()
	key := userID + "/" + id + "_" + filename
	url, err := s.store.Upload(ctx, key, contentType, data)
	if err != nil {
		return nil, err
	}
	media := &models.Media{
		ID: id, UserID: userID, Key: key, URL: url, Type: "file",
		Size: int64(len(data)), ContentType: contentType, CreatedAt: time.Now().UTC(),
	}
	if err := s.repo.Insert(ctx, media); err != nil {
		return nil, err
	}
	return media, nil
}

func (s *MediaService) GetByID(ctx context.Context, id string) (*models.Media, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *MediaService) UploadImage(ctx context.Context, userID string, filename string, contentType string, data []byte) (*models.Media, error) {
	// generate key
	id := utils.NewID()
	key := userID + "/" + id + "_" + filename

	// upload original
	url, err := s.store.Upload(ctx, key, contentType, data)
	if err != nil {
		return nil, err
	}

	thumbKey := key + "_thumb.jpg"
	thumbBytes, err := generateThumbnail(data)
	if err == nil {
		_, _ = s.store.Upload(ctx, thumbKey, "image/jpeg", thumbBytes)
	}

	media := &models.Media{
		ID:          id,
		UserID:      userID,
		Key:         key,
		URL:         url,
		Thumbnail:   thumbKey,
		Type:        "image",
		Size:        int64(len(data)),
		ContentType: contentType,
		CreatedAt:   time.Now().UTC(),
	}
	if err := s.repo.Insert(ctx, media); err != nil {
		return nil, err
	}
	return media, nil
}

func (s *MediaService) GetPresignedURL(ctx context.Context, mediaKey string) (string, error) {
	// optionally check cache
	url, err := s.store.PresignURL(ctx, mediaKey, s.presignTTL)
	if err != nil {
		return "", err
	}
	return url, nil
}

// helper
func generateThumbnail(data []byte) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	thumb := imaging.Resize(img, 320, 0, imaging.Lanczos)
	var buf bytes.Buffer
	if err := imaging.Encode(&buf, thumb, imaging.JPEG); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
