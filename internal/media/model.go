package media

import "errors"

// UploadType represents the category of an uploaded file.
type UploadType string

const (
	TypeAvatar UploadType = "avatar"
	TypeCover  UploadType = "cover"
)

// UploadResult is the response returned after a successful upload.
type UploadResult struct {
	URL string `json:"url"`
	Key string `json:"key"`
}

// File size limits (in bytes).
const (
	MaxAvatarSize = 2 << 20 // 2 MB
	MaxCoverSize  = 5 << 20 // 5 MB
)

// Allowed MIME types for image uploads.
var allowedImageTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
	"image/gif":  ".gif",
}

// Sentinel errors.
var (
	ErrFileTooLarge      = errors.New("file_too_large")
	ErrInvalidFileType   = errors.New("invalid_file_type")
	ErrInvalidUploadType = errors.New("invalid_upload_type")
	ErrUploadFailed      = errors.New("upload_failed")
)
