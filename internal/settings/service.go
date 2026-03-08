package settings

import (
	"context"

	"github.com/devsin/coreapi/internal/auth"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Service coordinates user settings operations.
type Service struct {
	repo *Repository
	log  *zap.Logger
}

func NewService(repo *Repository, log *zap.Logger) *Service {
	return &Service{repo: repo, log: log}
}

// GetOrCreateSettings ensures the user settings exist and returns them.
func (s *Service) GetOrCreateSettings(ctx context.Context, claims auth.UserClaims) (*UserSettingsDTO, error) {
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, err
	}

	settings, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Create default settings if not exists
	if settings == nil {
		settings, err = s.repo.CreateDefault(ctx, userID)
		if err != nil {
			return nil, err
		}
	}

	return settingsToDTO(settings), nil
}

// UpdateSettings updates the authenticated user's settings.
func (s *Service) UpdateSettings(ctx context.Context, claims auth.UserClaims, req UpdateSettingsRequest) (*UserSettingsDTO, error) {
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, err
	}

	// Get existing settings or create defaults
	existing, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if existing == nil {
		existing = &UserSettings{
			UserID:              userID,
			EmailLinkActivity:   true,
			EmailWeeklyDigest:   true,
			EmailProductUpdates: false,
			ProfilePublic:       true,
			ShowActivityStatus:  true,
			AllowNsfwContent:    false,
		}
	}

	// Apply updates
	if req.EmailLinkActivity != nil {
		existing.EmailLinkActivity = *req.EmailLinkActivity
	}
	if req.EmailWeeklyDigest != nil {
		existing.EmailWeeklyDigest = *req.EmailWeeklyDigest
	}
	if req.EmailProductUpdates != nil {
		existing.EmailProductUpdates = *req.EmailProductUpdates
	}
	if req.ProfilePublic != nil {
		existing.ProfilePublic = *req.ProfilePublic
	}
	if req.ShowActivityStatus != nil {
		existing.ShowActivityStatus = *req.ShowActivityStatus
	}
	if req.AllowNsfwContent != nil {
		existing.AllowNsfwContent = *req.AllowNsfwContent
	}

	updated, err := s.repo.Upsert(ctx, userID, existing)
	if err != nil {
		return nil, err
	}

	return settingsToDTO(updated), nil
}

func settingsToDTO(s *UserSettings) *UserSettingsDTO {
	return &UserSettingsDTO{
		EmailLinkActivity:   s.EmailLinkActivity,
		EmailWeeklyDigest:   s.EmailWeeklyDigest,
		EmailProductUpdates: s.EmailProductUpdates,
		ProfilePublic:       s.ProfilePublic,
		ShowActivityStatus:  s.ShowActivityStatus,
		AllowNsfwContent:    s.AllowNsfwContent,
		UpdatedAt:           s.UpdatedAt,
	}
}

// IsProfilePublic checks whether a user's profile is publicly visible.
// Returns true if settings don't exist (default is public).
func (s *Service) IsProfilePublic(ctx context.Context, userID uuid.UUID) (bool, error) {
	return s.repo.IsProfilePublic(ctx, userID)
}
