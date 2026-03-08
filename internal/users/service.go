package users

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/devsin/coreapi/common/reserved"
	"github.com/devsin/coreapi/internal/auth"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

var (
	ErrUsernameTaken    = errors.New("username_taken")
	ErrUsernameInvalid  = errors.New("username_invalid")
	ErrUsernameReserved = errors.New("username_reserved")
	ErrUserNotFound     = errors.New("user_not_found")
	ErrCannotFollowSelf = errors.New("cannot_follow_self")
	ErrProfilePrivate   = errors.New("profile_private")
)

// ProfilePrivacyChecker checks whether a user's profile is publicly visible.
type ProfilePrivacyChecker interface {
	IsProfilePublic(ctx context.Context, userID uuid.UUID) (bool, error)
}

// Notifier sends in-app notifications.
type Notifier interface {
	Notify(ctx context.Context, p NotifyParams)
}

// NotifyParams holds the parameters for sending a notification.
type NotifyParams struct {
	UserID     string
	ActorID    string
	Type       string
	EntityType string
	EntityID   string
	Title      string
	Body       string
	URL        string
}

var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{3,20}$`)

// Service coordinates user operations.
type Service struct {
	repo     *Repository
	log      *zap.Logger
	privacy  ProfilePrivacyChecker
	notifier Notifier
}

func NewService(repo *Repository, log *zap.Logger) *Service {
	return &Service{repo: repo, log: log}
}

// SetPrivacyChecker sets the optional profile privacy checker.
func (s *Service) SetPrivacyChecker(checker ProfilePrivacyChecker) {
	s.privacy = checker
}

// SetNotifier sets the optional notifier for in-app notifications.
func (s *Service) SetNotifier(n Notifier) {
	s.notifier = n
}

// isProfileVisibleTo returns true if the profile owner's profile is visible to the viewer.
// The owner can always see their own profile. If no privacy checker is set, profiles are public.
func (s *Service) isProfileVisibleTo(ctx context.Context, profileUserID uuid.UUID, viewerID *uuid.UUID) (bool, error) {
	// Owner always sees own profile
	if viewerID != nil && *viewerID == profileUserID {
		return true, nil
	}
	if s.privacy == nil {
		return true, nil
	}
	return s.privacy.IsProfilePublic(ctx, profileUserID)
}

// CheckUsernameAvailability validates a username and checks if it's available.
// Returns (available, reason, error).
func (s *Service) CheckUsernameAvailability(ctx context.Context, username string) (available bool, reason string, err error) {
	username = strings.ToLower(strings.TrimSpace(username))

	if !usernameRegex.MatchString(username) {
		return false, "invalid", nil
	}

	if reserved.IsReserved(username) {
		return false, "reserved", nil
	}

	taken, err := s.repo.IsUsernameTaken(ctx, username, uuid.Nil)
	if err != nil {
		return false, "", err
	}
	if taken {
		return false, "taken", nil
	}

	return true, "", nil
}

// GetOrCreateMe ensures the user exists and returns the DTO.
func (s *Service) GetOrCreateMe(ctx context.Context, claims auth.UserClaims) (*UserDTO, error) {
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, err
	}

	u, err := s.repo.CreateIfNotExists(ctx, userID)
	if err != nil {
		return nil, err
	}

	return userToDTO(u), nil
}

// UpdateMe updates the authenticated user's profile.
func (s *Service) UpdateMe(ctx context.Context, claims auth.UserClaims, req UpdateProfileRequest) (*UserDTO, error) {
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, err
	}

	// Validate and check username if provided
	if req.Username != nil {
		username := strings.ToLower(strings.TrimSpace(*req.Username))
		req.Username = &username

		if !usernameRegex.MatchString(username) {
			return nil, ErrUsernameInvalid
		}

		if reserved.IsReserved(username) {
			return nil, ErrUsernameReserved
		}

		taken, err := s.repo.IsUsernameTaken(ctx, username, userID)
		if err != nil {
			return nil, err
		}
		if taken {
			return nil, ErrUsernameTaken
		}
	}

	u, err := s.repo.UpdateUser(ctx, userID, UpdateUserParams(req))
	if err != nil {
		return nil, err
	}

	return userToDTO(u), nil
}

// UpdateProfileRequest is the input for updating a user's profile.
type UpdateProfileRequest struct {
	Username    *string `json:"username,omitempty"`
	DisplayName *string `json:"display_name,omitempty"`
	Bio         *string `json:"bio,omitempty"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
}

func userToDTO(u *User) *UserDTO {
	return &UserDTO{
		ID:          u.ID.String(),
		Username:    u.Username,
		DisplayName: u.DisplayName,
		Bio:         u.Bio,
		AvatarURL:   u.AvatarURL,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}

// FollowUser creates a follow relationship.
func (s *Service) FollowUser(ctx context.Context, followerID, followingID uuid.UUID) error {
	if followerID == followingID {
		return ErrCannotFollowSelf
	}

	// Check if target user exists
	target, err := s.repo.GetByID(ctx, followingID)
	if err != nil {
		return err
	}
	if target == nil {
		return ErrUserNotFound
	}

	if err := s.repo.FollowUser(ctx, followerID, followingID); err != nil {
		return err
	}

	// Fire-and-forget notification to the followed user.
	if s.notifier != nil {
		actorName := s.resolveActorName(ctx, followerID)
		go s.notifier.Notify(context.Background(), NotifyParams{ //nolint:contextcheck // intentional: fire-and-forget goroutine that outlives the request
			UserID:     followingID.String(),
			ActorID:    followerID.String(),
			Type:       "follow",
			EntityType: "user",
			EntityID:   followerID.String(),
			Title:      actorName + " started following you",
			URL:        "/p/" + s.resolveUsername(ctx, followerID),
		})
	}

	return nil
}

// UnfollowUser removes a follow relationship.
func (s *Service) UnfollowUser(ctx context.Context, followerID, followingID uuid.UUID) error {
	return s.repo.UnfollowUser(ctx, followerID, followingID)
}

// GetUserWithStats returns a user profile with follow stats and optional is_following flag.
func (s *Service) GetUserWithStats(ctx context.Context, username string, viewerID *uuid.UUID) (*UserWithStatsDTO, error) {
	u, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, ErrUserNotFound
	}

	// Check profile privacy
	visible, err := s.isProfileVisibleTo(ctx, u.ID, viewerID)
	if err != nil {
		return nil, err
	}
	if !visible {
		// Return minimal info with private flag
		return &UserWithStatsDTO{
			UserDTO: UserDTO{
				ID:          u.ID.String(),
				Username:    u.Username,
				DisplayName: u.DisplayName,
				AvatarURL:   u.AvatarURL,
				CreatedAt:   u.CreatedAt,
				UpdatedAt:   u.UpdatedAt,
			},
			IsPrivate: true,
		}, nil
	}

	followersCount, err := s.repo.GetFollowersCount(ctx, u.ID)
	if err != nil {
		return nil, err
	}

	followingCount, err := s.repo.GetFollowingCount(ctx, u.ID)
	if err != nil {
		return nil, err
	}

	var isFollowing bool
	if viewerID != nil {
		isFollowing, err = s.repo.IsFollowing(ctx, *viewerID, u.ID)
		if err != nil {
			return nil, err
		}
	}

	return &UserWithStatsDTO{
		UserDTO:        *userToDTO(u),
		FollowersCount: followersCount,
		FollowingCount: followingCount,
		IsFollowing:    isFollowing,
	}, nil
}

// GetFollowers returns paginated followers for a user.
func (s *Service) GetFollowers(ctx context.Context, username string, currentUserID *uuid.UUID, limit, offset int32) (*FollowListResponse, error) {
	u, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, ErrUserNotFound
	}

	// Check profile privacy
	visible, err := s.isProfileVisibleTo(ctx, u.ID, currentUserID)
	if err != nil {
		return nil, err
	}
	if !visible {
		return nil, ErrProfilePrivate
	}

	total, err := s.repo.GetFollowersCount(ctx, u.ID)
	if err != nil {
		return nil, err
	}

	followers, err := s.repo.GetFollowers(ctx, u.ID, limit, offset)
	if err != nil {
		return nil, err
	}

	dtos := make([]*UserDTO, len(followers))
	for i, f := range followers {
		dto := userToDTO(f)
		// Check if current user is following each follower
		if currentUserID != nil {
			isFollowing, err := s.repo.IsFollowing(ctx, *currentUserID, f.ID)
			if err == nil {
				dto.IsFollowing = &isFollowing
			}
		}
		dtos[i] = dto
	}

	return &FollowListResponse{
		Users:   dtos,
		Total:   total,
		HasMore: int64(offset)+int64(len(followers)) < total,
	}, nil
}

// GetFollowing returns paginated list of users that a user is following.
func (s *Service) GetFollowing(ctx context.Context, username string, currentUserID *uuid.UUID, limit, offset int32) (*FollowListResponse, error) {
	u, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, ErrUserNotFound
	}

	// Check profile privacy
	visible, err := s.isProfileVisibleTo(ctx, u.ID, currentUserID)
	if err != nil {
		return nil, err
	}
	if !visible {
		return nil, ErrProfilePrivate
	}

	total, err := s.repo.GetFollowingCount(ctx, u.ID)
	if err != nil {
		return nil, err
	}

	following, err := s.repo.GetFollowing(ctx, u.ID, limit, offset)
	if err != nil {
		return nil, err
	}

	dtos := make([]*UserDTO, len(following))
	for i, f := range following {
		dto := userToDTO(f)
		// Check if current user is following each user
		if currentUserID != nil {
			isFollowing, err := s.repo.IsFollowing(ctx, *currentUserID, f.ID)
			if err == nil {
				dto.IsFollowing = &isFollowing
			}
		}
		dtos[i] = dto
	}

	return &FollowListResponse{
		Users:   dtos,
		Total:   total,
		HasMore: int64(offset)+int64(len(following)) < total,
	}, nil
}

// DiscoverUsers returns a paginated list of users for the explore page.
func (s *Service) DiscoverUsers(ctx context.Context, sort string, currentUserID *uuid.UUID, limit, offset int32) (*DiscoverUsersResponse, error) {
	var users []*DiscoverUser
	var err error

	if sort == "new" {
		users, err = s.repo.DiscoverUsersNew(ctx, limit, offset, currentUserID)
	} else {
		users, err = s.repo.DiscoverUsersPopular(ctx, limit, offset, currentUserID)
	}
	if err != nil {
		return nil, err
	}

	total, err := s.repo.CountDiscoverUsers(ctx, currentUserID)
	if err != nil {
		return nil, err
	}

	dtos := make([]*DiscoverUserDTO, len(users))
	for i, u := range users {
		dto := &DiscoverUserDTO{
			ID:            u.ID.String(),
			Username:      u.Username,
			DisplayName:   u.DisplayName,
			Bio:           u.Bio,
			AvatarURL:     u.AvatarURL,
			CreatedAt:     u.CreatedAt,
			UpdatedAt:     u.UpdatedAt,
			FollowerCount: u.FollowerCount,
		}
		if currentUserID != nil {
			isFollowing, fErr := s.repo.IsFollowing(ctx, *currentUserID, u.ID)
			if fErr == nil {
				dto.IsFollowing = isFollowing
			}
		}
		dtos[i] = dto
	}

	return &DiscoverUsersResponse{
		Users:   dtos,
		Total:   total,
		HasMore: int64(offset)+int64(len(users)) < total,
	}, nil
}

// resolveActorName returns a display name for notification titles.
func (s *Service) resolveActorName(ctx context.Context, userID uuid.UUID) string {
	u, err := s.repo.GetByID(ctx, userID)
	if err != nil || u == nil {
		return "Someone"
	}
	if u.DisplayName != nil && *u.DisplayName != "" {
		return *u.DisplayName
	}
	if u.Username != nil && *u.Username != "" {
		return "@" + *u.Username
	}
	return "Someone"
}

// resolveUsername returns the username string for building profile URLs.
func (s *Service) resolveUsername(ctx context.Context, userID uuid.UUID) string {
	u, err := s.repo.GetByID(ctx, userID)
	if err != nil || u == nil || u.Username == nil {
		return userID.String()
	}
	return *u.Username
}
