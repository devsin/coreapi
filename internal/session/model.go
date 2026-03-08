package session

import (
	"errors"
	"time"
)

var ErrCannotDeleteCurrentSession = errors.New("cannot_delete_current_session")

// DTO is the API response for a single session.
type DTO struct {
	ID           string `json:"id"`
	SessionID    string `json:"session_id"`
	IPAddress    string `json:"ip_address,omitempty"`
	Browser      string `json:"browser,omitempty"`
	OS           string `json:"os,omitempty"`
	DeviceType   string `json:"device_type,omitempty"`
	IsCurrent    bool   `json:"is_current"`
	LastActiveAt string `json:"last_active_at"`
	CreatedAt    string `json:"created_at"`
}

// ListResponse wraps a list of sessions.
type ListResponse struct {
	Sessions []DTO `json:"sessions"`
	Total    int   `json:"total"`
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}
