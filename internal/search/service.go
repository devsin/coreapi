package search

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	dbgen "github.com/devsin/coreapi/gen/db"
	"go.uber.org/zap"
)

// userLookup provides methods to look up user relationships.
type userLookup interface {
	IsFollowing(ctx context.Context, followerID, followingID uuid.UUID) (bool, error)
}

// userRepoAdapter adapts the database queries to userLookup.
type userRepoAdapter struct {
	q *dbgen.Queries
}

func newUserRepoAdapter(pool *pgxpool.Pool) *userRepoAdapter {
	return &userRepoAdapter{q: dbgen.New(pool)}
}

func (a *userRepoAdapter) IsFollowing(ctx context.Context, followerID, followingID uuid.UUID) (bool, error) {
	return a.q.IsFollowing(ctx, dbgen.IsFollowingParams{
		FollowerID:  followerID,
		FollowingID: followingID,
	})
}

const (
	defaultLimit = 10
	maxLimit     = 50
)

// Service coordinates search operations.
type Service struct {
	repo  *Repository
	users userLookup
	log   *zap.Logger
}

// NewService creates a search service.
func NewService(pool *pgxpool.Pool, log *zap.Logger) *Service {
	return &Service{
		repo:  NewRepository(pool),
		users: newUserRepoAdapter(pool),
		log:   log,
	}
}

// Search performs a search across users.
func (s *Service) Search(ctx context.Context, query string, limit, offset int32, viewerID *uuid.UUID) (*Response, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return &Response{Query: query}, nil
	}

	if limit <= 0 || limit > maxLimit {
		limit = defaultLimit
	}
	if offset < 0 {
		offset = 0
	}

	users, err := s.searchUsers(ctx, query, limit, offset, viewerID)
	if err != nil {
		return nil, err
	}

	return &Response{Query: query, Users: users}, nil
}

func (s *Service) searchUsers(ctx context.Context, query string, limit, offset int32, viewerID *uuid.UUID) (*PaginatedResult[UserResult], error) {
	rows, total, err := s.repo.SearchUsers(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}

	items := make([]UserResult, 0, len(rows))
	for _, r := range rows {
		u := UserResult{
			ID:            r.ID.String(),
			Username:      r.Username,
			DisplayName:   r.DisplayName,
			Bio:           r.Bio,
			AvatarURL:     r.AvatarUrl,
			FollowerCount: r.FollowerCount,
		}

		if viewerID != nil {
			isFollowing, err := s.users.IsFollowing(ctx, *viewerID, r.ID)
			if err == nil {
				u.IsFollowing = isFollowing
			}
		}

		items = append(items, u)
	}

	return &PaginatedResult[UserResult]{
		Items:   items,
		Total:   total,
		HasMore: int64(offset)+int64(limit) < total,
	}, nil
}
