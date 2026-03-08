package media

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/devsin/coreapi/internal/storage"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Service handles media upload business logic.
type Service struct {
	store storage.ObjectStore
	log   *zap.Logger
}

// NewService creates a new media service.
func NewService(store storage.ObjectStore, log *zap.Logger) *Service {
	return &Service{store: store, log: log}
}

// Upload validates and stores a file, returning the public URL and storage key.
func (s *Service) Upload(
	ctx context.Context,
	userID uuid.UUID,
	uploadType UploadType,
	file io.Reader,
	contentType string,
	size int64,
) (*UploadResult, error) {
	// Validate upload type.
	maxSize, err := s.maxSizeForType(uploadType)
	if err != nil {
		return nil, err
	}

	// Validate file size.
	if size > maxSize {
		return nil, ErrFileTooLarge
	}

	// Validate content type.
	ext, ok := allowedImageTypes[contentType]
	if !ok {
		return nil, ErrInvalidFileType
	}

	// Generate a unique storage key: {type}/{userID}/{timestamp}-{uuid}{ext}
	key := fmt.Sprintf("%ss/%s/%d-%s%s",
		uploadType,
		userID.String(),
		time.Now().UnixMilli(),
		uuid.New().String()[:8],
		ext,
	)

	// Upload to object store.
	url, err := s.store.Upload(ctx, key, file, contentType)
	if err != nil {
		s.log.Error("media upload failed",
			zap.String("type", string(uploadType)),
			zap.String("user_id", userID.String()),
			zap.Error(err),
		)
		return nil, ErrUploadFailed
	}

	s.log.Info("media uploaded",
		zap.String("type", string(uploadType)),
		zap.String("key", key),
		zap.String("user_id", userID.String()),
		zap.Int64("size", size),
	)

	return &UploadResult{URL: url, Key: key}, nil
}

// Delete removes a previously uploaded file by its storage key.
func (s *Service) Delete(ctx context.Context, key string) error {
	if err := s.store.Delete(ctx, key); err != nil {
		s.log.Error("media delete failed", zap.String("key", key), zap.Error(err))
		return ErrUploadFailed
	}
	return nil
}

func (s *Service) maxSizeForType(t UploadType) (int64, error) {
	switch t {
	case TypeAvatar:
		return MaxAvatarSize, nil
	case TypeCover:
		return MaxCoverSize, nil
	default:
		return 0, ErrInvalidUploadType
	}
}
