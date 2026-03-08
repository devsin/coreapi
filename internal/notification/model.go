package notification

import "time"

// Notification types.
const (
	TypeFollow = "follow"
)

// Entity types.
const (
	EntityUser = "user"
)

// DTO is the API response for a notification.
type DTO struct {
	ID         string    `json:"id"`
	ActorID    *string   `json:"actor_id,omitempty"`
	Type       string    `json:"type"`
	EntityType *string   `json:"entity_type,omitempty"`
	EntityID   *string   `json:"entity_id,omitempty"`
	Title      string    `json:"title"`
	Body       *string   `json:"body,omitempty"`
	URL        *string   `json:"url,omitempty"`
	IsRead     bool      `json:"is_read"`
	ReadAt     *string   `json:"read_at,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	// Enriched fields
	Actor *ActorDTO `json:"actor,omitempty"`
}

// ActorDTO is a minimal user representation for the notification actor.
type ActorDTO struct {
	ID          string  `json:"id"`
	Username    *string `json:"username,omitempty"`
	DisplayName *string `json:"display_name,omitempty"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
}

// ListResponse wraps a paginated list of notifications.
type ListResponse struct {
	Notifications []DTO `json:"notifications"`
	Total         int64 `json:"total"`
	HasMore       bool  `json:"has_more"`
}

// UnreadCountResponse wraps the unread count.
type UnreadCountResponse struct {
	Count int64 `json:"count"`
}

// CreateParams holds the input for creating a notification.
type CreateParams struct {
	UserID     string
	ActorID    string
	Type       string
	EntityType string
	EntityID   string
	Title      string
	Body       string
	URL        string
}
