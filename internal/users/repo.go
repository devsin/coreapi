package users

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	dbgen "github.com/devsin/coreapi/gen/db"
)

// Repository encapsulates database operations for users.
type Repository struct {
	pool *pgxpool.Pool
	q    *dbgen.Queries
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool, q: dbgen.New(pool)}
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	row, err := r.q.GetUser(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil //nolint:nilnil // not found is expressed as (nil, nil)
		}
		return nil, err
	}

	return rowToUser(row.ID, row.Username, row.DisplayName, row.Bio, row.AvatarUrl, row.CreatedAt, row.UpdatedAt), nil
}

func (r *Repository) GetByUsername(ctx context.Context, username string) (*User, error) {
	row, err := r.q.GetUserByUsername(ctx, &username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil //nolint:nilnil // not found is expressed as (nil, nil)
		}
		return nil, err
	}

	return rowToUser(row.ID, row.Username, row.DisplayName, row.Bio, row.AvatarUrl, row.CreatedAt, row.UpdatedAt), nil
}

func (r *Repository) CreateIfNotExists(ctx context.Context, id uuid.UUID) (*User, error) {
	row, err := r.q.InsertUser(ctx, id)
	if err != nil {
		return nil, err
	}

	return rowToUser(row.ID, row.Username, row.DisplayName, row.Bio, row.AvatarUrl, row.CreatedAt, row.UpdatedAt), nil
}

// UpdateUser updates user profile fields and returns the updated user.
func (r *Repository) UpdateUser(ctx context.Context, id uuid.UUID, updates UpdateUserParams) (*User, error) {
	row, err := r.q.UpdateUser(ctx, dbgen.UpdateUserParams{
		ID:          id,
		Username:    updates.Username,
		DisplayName: updates.DisplayName,
		Bio:         updates.Bio,
		AvatarUrl:   updates.AvatarURL,
	})
	if err != nil {
		return nil, err
	}
	return rowToUser(row.ID, row.Username, row.DisplayName, row.Bio, row.AvatarUrl, row.CreatedAt, row.UpdatedAt), nil
}

// IsUsernameTaken checks if username is taken by another user.
func (r *Repository) IsUsernameTaken(ctx context.Context, username string, excludeUserID uuid.UUID) (bool, error) {
	return r.q.IsUsernameTaken(ctx, dbgen.IsUsernameTakenParams{
		Username: &username,
		ID:       excludeUserID,
	})
}

// UpdateUserParams contains optional fields for updating a user.
type UpdateUserParams struct {
	Username    *string
	DisplayName *string
	Bio         *string
	AvatarURL   *string
}

// rowToUser converts any sqlc-generated user row (which all share the same
// core columns) to the internal User model.
func rowToUser(id uuid.UUID, username, displayName, bio, avatarURL *string, createdAt, updatedAt time.Time) *User {
	return &User{
		ID:          id,
		Username:    username,
		DisplayName: displayName,
		Bio:         bio,
		AvatarURL:   avatarURL,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
}

// FollowUser creates a follow relationship.
func (r *Repository) FollowUser(ctx context.Context, followerID, followingID uuid.UUID) error {
	return r.q.FollowUser(ctx, dbgen.FollowUserParams{
		FollowerID:  followerID,
		FollowingID: followingID,
	})
}

// UnfollowUser removes a follow relationship.
func (r *Repository) UnfollowUser(ctx context.Context, followerID, followingID uuid.UUID) error {
	return r.q.UnfollowUser(ctx, dbgen.UnfollowUserParams{
		FollowerID:  followerID,
		FollowingID: followingID,
	})
}

// IsFollowing checks if followerID follows followingID.
func (r *Repository) IsFollowing(ctx context.Context, followerID, followingID uuid.UUID) (bool, error) {
	return r.q.IsFollowing(ctx, dbgen.IsFollowingParams{
		FollowerID:  followerID,
		FollowingID: followingID,
	})
}

// GetFollowersCount returns the number of followers for a user.
func (r *Repository) GetFollowersCount(ctx context.Context, userID uuid.UUID) (int64, error) {
	return r.q.GetFollowersCount(ctx, userID)
}

// GetFollowingCount returns the number of users a user is following.
func (r *Repository) GetFollowingCount(ctx context.Context, userID uuid.UUID) (int64, error) {
	return r.q.GetFollowingCount(ctx, userID)
}

// GetFollowers returns paginated list of followers for a user.
func (r *Repository) GetFollowers(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]*User, error) {
	rows, err := r.q.GetFollowers(ctx, dbgen.GetFollowersParams{
		FollowingID: userID,
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		return nil, err
	}

	users := make([]*User, len(rows))
	for i, row := range rows {
		users[i] = &User{
			ID:          row.ID,
			Username:    row.Username,
			DisplayName: row.DisplayName,
			Bio:         row.Bio,
			AvatarURL:   row.AvatarUrl,
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		}
	}
	return users, nil
}

// GetFollowing returns paginated list of users that a user is following.
func (r *Repository) GetFollowing(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]*User, error) {
	rows, err := r.q.GetFollowing(ctx, dbgen.GetFollowingParams{
		FollowerID: userID,
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		return nil, err
	}

	users := make([]*User, len(rows))
	for i, row := range rows {
		users[i] = &User{
			ID:          row.ID,
			Username:    row.Username,
			DisplayName: row.DisplayName,
			Bio:         row.Bio,
			AvatarURL:   row.AvatarUrl,
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		}
	}
	return users, nil
}

// DiscoverUser is an intermediate model with follower count.
type DiscoverUser struct {
	User
	FollowerCount int64
}

// uuidToPgtype converts a *uuid.UUID to pgtype.UUID (null-safe).
func uuidToPgtype(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: *id, Valid: true}
}

// DiscoverUsersPopular returns users ordered by follower count.
func (r *Repository) DiscoverUsersPopular(ctx context.Context, limit, offset int32, excludeUserID *uuid.UUID) ([]*DiscoverUser, error) {
	rows, err := r.q.DiscoverUsersPopular(ctx, dbgen.DiscoverUsersPopularParams{
		Limit:         limit,
		Offset:        offset,
		ExcludeUserID: uuidToPgtype(excludeUserID),
	})
	if err != nil {
		return nil, err
	}
	users := make([]*DiscoverUser, len(rows))
	for i, row := range rows {
		users[i] = &DiscoverUser{
			User: User{
				ID:          row.ID,
				Username:    row.Username,
				DisplayName: row.DisplayName,
				Bio:         row.Bio,
				AvatarURL:   row.AvatarUrl,
				CreatedAt:   row.CreatedAt,
				UpdatedAt:   row.UpdatedAt,
			},
			FollowerCount: row.FollowerCount,
		}
	}
	return users, nil
}

// DiscoverUsersNew returns users ordered by creation date.
func (r *Repository) DiscoverUsersNew(ctx context.Context, limit, offset int32, excludeUserID *uuid.UUID) ([]*DiscoverUser, error) {
	rows, err := r.q.DiscoverUsersNew(ctx, dbgen.DiscoverUsersNewParams{
		Limit:         limit,
		Offset:        offset,
		ExcludeUserID: uuidToPgtype(excludeUserID),
	})
	if err != nil {
		return nil, err
	}
	users := make([]*DiscoverUser, len(rows))
	for i, row := range rows {
		users[i] = &DiscoverUser{
			User: User{
				ID:          row.ID,
				Username:    row.Username,
				DisplayName: row.DisplayName,
				Bio:         row.Bio,
				AvatarURL:   row.AvatarUrl,
				CreatedAt:   row.CreatedAt,
				UpdatedAt:   row.UpdatedAt,
			},
			FollowerCount: row.FollowerCount,
		}
	}
	return users, nil
}

// CountDiscoverUsers returns the total count of users with usernames.
func (r *Repository) CountDiscoverUsers(ctx context.Context, excludeUserID *uuid.UUID) (int64, error) {
	return r.q.CountDiscoverUsers(ctx, uuidToPgtype(excludeUserID))
}
