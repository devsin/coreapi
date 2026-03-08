package settings

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	dbgen "github.com/devsin/coreapi/gen/db"
)

// Repository encapsulates database operations for user settings.
type Repository struct {
	pool *pgxpool.Pool
	q    *dbgen.Queries
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool, q: dbgen.New(pool)}
}

func (r *Repository) GetByUserID(ctx context.Context, userID uuid.UUID) (*UserSettings, error) {
	row, err := r.q.GetUserSettings(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil //nolint:nilnil // not found is expressed as (nil, nil)
		}
		return nil, err
	}

	return &UserSettings{
		UserID:              row.UserID,
		EmailLinkActivity:   row.EmailLinkActivity,
		EmailWeeklyDigest:   row.EmailWeeklyDigest,
		EmailProductUpdates: row.EmailProductUpdates,
		ProfilePublic:       row.ProfilePublic,
		ShowActivityStatus:  row.ShowActivityStatus,
		AllowNsfwContent:    row.AllowNsfwContent,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}, nil
}

func (r *Repository) CreateDefault(ctx context.Context, userID uuid.UUID) (*UserSettings, error) {
	row, err := r.q.CreateDefaultUserSettings(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &UserSettings{
		UserID:              row.UserID,
		EmailLinkActivity:   row.EmailLinkActivity,
		EmailWeeklyDigest:   row.EmailWeeklyDigest,
		EmailProductUpdates: row.EmailProductUpdates,
		ProfilePublic:       row.ProfilePublic,
		ShowActivityStatus:  row.ShowActivityStatus,
		AllowNsfwContent:    row.AllowNsfwContent,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}, nil
}

func (r *Repository) Upsert(ctx context.Context, userID uuid.UUID, s *UserSettings) (*UserSettings, error) {
	row, err := r.q.UpsertUserSettings(ctx, dbgen.UpsertUserSettingsParams{
		UserID:              userID,
		EmailLinkActivity:   s.EmailLinkActivity,
		EmailWeeklyDigest:   s.EmailWeeklyDigest,
		EmailProductUpdates: s.EmailProductUpdates,
		ProfilePublic:       s.ProfilePublic,
		ShowActivityStatus:  s.ShowActivityStatus,
		AllowNsfwContent:    s.AllowNsfwContent,
	})
	if err != nil {
		return nil, err
	}

	return &UserSettings{
		UserID:              row.UserID,
		EmailLinkActivity:   row.EmailLinkActivity,
		EmailWeeklyDigest:   row.EmailWeeklyDigest,
		EmailProductUpdates: row.EmailProductUpdates,
		ProfilePublic:       row.ProfilePublic,
		ShowActivityStatus:  row.ShowActivityStatus,
		AllowNsfwContent:    row.AllowNsfwContent,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}, nil
}

// IsProfilePublic checks whether a user's profile is public.
// Returns true if settings don't exist (default is public).
func (r *Repository) IsProfilePublic(ctx context.Context, userID uuid.UUID) (bool, error) {
	s, err := r.GetByUserID(ctx, userID)
	if err != nil {
		return true, err
	}
	if s == nil {
		return true, nil // default is public
	}
	return s.ProfilePublic, nil
}
