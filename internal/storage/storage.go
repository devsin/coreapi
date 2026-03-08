// Package storage provides object storage abstractions and implementations.
package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// ObjectStore abstracts object storage operations (S3, R2, etc.).
type ObjectStore interface {
	// Upload stores an object and returns its public URL.
	Upload(ctx context.Context, key string, body io.Reader, contentType string) (string, error)

	// Delete removes an object by key.
	Delete(ctx context.Context, key string) error

	// PublicURL returns the full public URL for a given object key.
	PublicURL(key string) string
}

// R2Config holds Cloudflare R2 connection settings.
type R2Config struct {
	AccountID       string
	AccessKeyID     string
	AccessKeySecret string
	Bucket          string
	PublicURL       string // CDN / public URL prefix, e.g. "https://uploads.example.com"
}

// R2Store implements ObjectStore using Cloudflare R2 (S3-compatible).
type R2Store struct {
	client    *s3.Client
	bucket    string
	publicURL string
}

// NewR2Store creates an R2-backed ObjectStore.
func NewR2Store(cfg R2Config) *R2Store {
	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.AccountID)

	client := s3.New(s3.Options{
		Region:       "auto",
		BaseEndpoint: aws.String(endpoint),
		Credentials:  credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.AccessKeySecret, ""),
	})

	return &R2Store{
		client:    client,
		bucket:    cfg.Bucket,
		publicURL: cfg.PublicURL,
	}
}

// Upload stores an object in R2 and returns its public URL.
func (s *R2Store) Upload(ctx context.Context, key string, body io.Reader, contentType string) (string, error) {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("r2 upload %q: %w", key, err)
	}
	return s.PublicURL(key), nil
}

// Delete removes an object from R2.
func (s *R2Store) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("r2 delete %q: %w", key, err)
	}
	return nil
}

// PublicURL returns the full public URL for a given object key.
func (s *R2Store) PublicURL(key string) string {
	return s.publicURL + "/" + key
}
