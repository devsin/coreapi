package search

// UserResult is a user returned from a search query.
type UserResult struct {
	ID            string  `json:"id"`
	Username      *string `json:"username,omitempty"`
	DisplayName   *string `json:"display_name,omitempty"`
	Bio           *string `json:"bio,omitempty"`
	AvatarURL     *string `json:"avatar_url,omitempty"`
	FollowerCount int64   `json:"follower_count"`
	IsFollowing   bool    `json:"is_following,omitempty"`
}

// Response is the search response.
type Response struct {
	Query string                       `json:"query"`
	Users *PaginatedResult[UserResult] `json:"users,omitempty"`
}

// PaginatedResult wraps a slice of results with pagination metadata.
type PaginatedResult[T any] struct {
	Items   []T   `json:"items"`
	Total   int64 `json:"total"`
	HasMore bool  `json:"has_more"`
}
