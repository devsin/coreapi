package settings

import (
	"time"

	"github.com/google/uuid"
)

// UserSettings is the persistence model for user settings.
type UserSettings struct {
	UserID              uuid.UUID `json:"user_id"`
	EmailLinkActivity   bool      `json:"email_link_activity"`
	EmailWeeklyDigest   bool      `json:"email_weekly_digest"`
	EmailProductUpdates bool      `json:"email_product_updates"`
	ProfilePublic       bool      `json:"profile_public"`
	ShowActivityStatus  bool      `json:"show_activity_status"`
	AllowNsfwContent    bool      `json:"allow_nsfw_content"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// UserSettingsDTO is the shape returned over the API.
type UserSettingsDTO struct {
	EmailLinkActivity   bool      `json:"email_link_activity"`
	EmailWeeklyDigest   bool      `json:"email_weekly_digest"`
	EmailProductUpdates bool      `json:"email_product_updates"`
	ProfilePublic       bool      `json:"profile_public"`
	ShowActivityStatus  bool      `json:"show_activity_status"`
	AllowNsfwContent    bool      `json:"allow_nsfw_content"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// UpdateSettingsRequest is the input for updating user settings.
type UpdateSettingsRequest struct {
	EmailLinkActivity   *bool `json:"email_link_activity,omitempty"`
	EmailWeeklyDigest   *bool `json:"email_weekly_digest,omitempty"`
	EmailProductUpdates *bool `json:"email_product_updates,omitempty"`
	ProfilePublic       *bool `json:"profile_public,omitempty"`
	ShowActivityStatus  *bool `json:"show_activity_status,omitempty"`
	AllowNsfwContent    *bool `json:"allow_nsfw_content,omitempty"`
}
