package storage

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/presign"
)

type S3Store struct {
	client    *s3.Client
	uploader  *manager.Uploader
	bucket    string
	region    string
	endpoint  string
	publicRead bool
}

func NewS3Store(ctx context.Context, region, bucket, endpoint string, publicRead bool) (*S3Store, error) {
	// support custom endpoint (MinIO) by setting AWS_REGION and using EndpointResolver if endpoint != ""
	cfg, err := awscfg.LoadDefaultConfig(ctx, awscfg.WithRegion(region))
	if err != nil { return nil, err }

	client := s3.NewFromConfig(cfg)
	uploader := manager.NewUploader(client)
	return &S3Store{client: client, uploader: uploader, bucket: bucket, region: region, endpoint: endpoint, publicRead: publicRead}, nil
}

func (s *S3Store) Upload(ctx context.Context, key string, contentType string, data []byte) (string, error) {
	_, err := s.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	if err != nil { return "", err }

	// return public URL if publicRead else empty (signed URL used)
	if s.publicRead {
		escaped := url.PathEscape(key)
		return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucket, s.region, escaped), nil
	}
	return "", nil
}

func (s *S3Store) PresignURL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	p := presign.NewPresignClient(s.client)
	req, err := p.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, presign.WithExpires(ttl))
	if err != nil { return "", err }
	return req.URL, nil
}
