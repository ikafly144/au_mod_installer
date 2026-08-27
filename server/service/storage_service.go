package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type StorageConfig struct {
	Endpoint        string // e.g. "http://minio:9000" or "https://<account>.r2.cloudflarestorage.com"
	PublicEndpoint  string // e.g. "http://localhost:9000" or "https://cdn.example.com"
	Region          string // e.g. "auto" or "us-east-1"
	Bucket          string // e.g. "au-mods"
	AccessKeyID     string
	SecretAccessKey string
	UsePathStyle    bool // true for MinIO and R2 path-style
}

type StorageService interface {
	Upload(ctx context.Context, key string, data []byte, contentType string) (string, error)
	UploadReader(ctx context.Context, key string, body io.Reader, size int64, contentType string) (string, error)
	Download(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
	GetPublicURL(key string) string
	EnsureBucket(ctx context.Context) error
}

type s3StorageService struct {
	client *s3.Client
	cfg    StorageConfig
}

func NewS3StorageService(ctx context.Context, cfg StorageConfig) (StorageService, error) {
	if cfg.Bucket == "" {
		return nil, errors.New("S3 bucket name is required")
	}
	if cfg.Region == "" {
		cfg.Region = "us-east-1"
	}

	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.Region),
	}

	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
		o.UsePathStyle = cfg.UsePathStyle
	})

	return &s3StorageService{
		client: client,
		cfg:    cfg,
	}, nil
}

func (s *s3StorageService) EnsureBucket(ctx context.Context) error {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s.cfg.Bucket),
	})
	if err != nil {
		slog.InfoContext(ctx, "Bucket does not exist, creating bucket", "bucket", s.cfg.Bucket)
		_, createErr := s.client.CreateBucket(ctx, &s3.CreateBucketInput{
			Bucket: aws.String(s.cfg.Bucket),
		})
		if createErr != nil {
			// Bucket might already exist or permission error
			var bErr *s3types.BucketAlreadyOwnedByYou
			if errors.As(createErr, &bErr) {
				return nil
			}
			return fmt.Errorf("failed to create bucket %s: %w", s.cfg.Bucket, createErr)
		}
		slog.InfoContext(ctx, "Created bucket successfully", "bucket", s.cfg.Bucket)
	}
	return nil
}

func (s *s3StorageService) Upload(ctx context.Context, key string, data []byte, contentType string) (string, error) {
	reader := bytes.NewReader(data)
	return s.UploadReader(ctx, key, reader, int64(len(data)), contentType)
}

func (s *s3StorageService) UploadReader(ctx context.Context, key string, body io.Reader, size int64, contentType string) (string, error) {
	key = strings.TrimPrefix(key, "/")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	input := &s3.PutObjectInput{
		Bucket:        aws.String(s.cfg.Bucket),
		Key:           aws.String(key),
		Body:          body,
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(size),
	}

	_, err := s.client.PutObject(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to upload object %s to bucket %s: %w", key, s.cfg.Bucket, err)
	}

	return s.GetPublicURL(key), nil
}

func (s *s3StorageService) Download(ctx context.Context, key string) ([]byte, error) {
	key = strings.TrimPrefix(key, "/")
	input := &s3.GetObjectInput{
		Bucket: aws.String(s.cfg.Bucket),
		Key:    aws.String(key),
	}

	output, err := s.client.GetObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get object %s from bucket %s: %w", key, s.cfg.Bucket, err)
	}
	defer output.Body.Close()

	return io.ReadAll(output.Body)
}

func (s *s3StorageService) Delete(ctx context.Context, key string) error {
	key = strings.TrimPrefix(key, "/")
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(s.cfg.Bucket),
		Key:    aws.String(key),
	}

	_, err := s.client.DeleteObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete object %s from bucket %s: %w", key, s.cfg.Bucket, err)
	}
	return nil
}

func escapeKeyPath(key string) string {
	parts := strings.Split(key, "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	return strings.Join(parts, "/")
}

func (s *s3StorageService) GetPublicURL(key string) string {
	key = strings.TrimPrefix(key, "/")
	escapedKey := escapeKeyPath(key)
	base := strings.TrimRight(s.cfg.PublicEndpoint, "/")
	if base == "" {
		base = strings.TrimRight(s.cfg.Endpoint, "/")
	}

	if base != "" {
		if s.cfg.UsePathStyle {
			return fmt.Sprintf("%s/%s/%s", base, s.cfg.Bucket, escapedKey)
		}
		// Virtual host style
		parsed, err := url.Parse(base)
		if err == nil {
			return fmt.Sprintf("%s://%s.%s/%s", parsed.Scheme, s.cfg.Bucket, parsed.Host, escapedKey)
		}
		return fmt.Sprintf("%s/%s/%s", base, s.cfg.Bucket, escapedKey)
	}

	// Default fallback to AWS S3 format
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.cfg.Bucket, s.cfg.Region, escapedKey)
}
