package users

import (
	"time"

	"github.com/google/uuid"
)

// User is the persistence model for users.
type User struct {
	ID          uuid.UUID `json:"id"`
	Username    *string   `json:"username"`
	DisplayName *string   `json:"display_name"`
	Bio         *string   `json:"bio"`
	AvatarURL   *string   `json:"avatar_url"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// UserDTO is the shape returned over the API.
type UserDTO struct {
	ID          string    `json:"id"`
	Username    *string   `json:"username,omitempty"`
	DisplayName *string   `json:"display_name,omitempty"`
	Bio         *string   `json:"bio,omitempty"`
	AvatarURL   *string   `json:"avatar_url,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	IsFollowing *bool     `json:"is_following,omitempty"`
}

// UserWithStatsDTO includes follow counts and follow status.
type UserWithStatsDTO struct {
	UserDTO
	FollowersCount int64 `json:"followers_count"`
	FollowingCount int64 `json:"following_count"`
	IsFollowing    bool  `json:"is_following,omitempty"`
	IsPrivate      bool  `json:"is_private,omitempty"`
}

// FollowListResponse is the paginated response for followers/following lists.
type FollowListResponse struct {
	Users   []*UserDTO `json:"users"`
	Total   int64      `json:"total"`
	HasMore bool       `json:"has_more"`
}

// DiscoverUserDTO is a user with stats for the discover/explore page.
type DiscoverUserDTO struct {
	ID            string    `json:"id"`
	Username      *string   `json:"username,omitempty"`
	DisplayName   *string   `json:"display_name,omitempty"`
	Bio           *string   `json:"bio,omitempty"`
	AvatarURL     *string   `json:"avatar_url,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	FollowerCount int64     `json:"follower_count"`
	IsFollowing   bool      `json:"is_following,omitempty"`
}

// DiscoverUsersResponse is the paginated response for discover users.
type DiscoverUsersResponse struct {
	Users   []*DiscoverUserDTO `json:"users"`
	Total   int64              `json:"total"`
	HasMore bool               `json:"has_more"`
}
