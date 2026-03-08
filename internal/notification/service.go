package notification

import (
	"context"

	dbgen "github.com/devsin/coreapi/gen/db"
	"github.com/devsin/coreapi/internal/users"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// Service coordinates notification operations.
type Service struct {
	repo     *Repository
	userRepo *users.Repository
	hub      *Hub
	log      *zap.Logger
}

// NewService creates a new notification service.
func NewService(repo *Repository, userRepo *users.Repository, hub *Hub, log *zap.Logger) *Service {
	return &Service{repo: repo, userRepo: userRepo, hub: hub, log: log}
}

// List returns paginated notifications for a user, enriched with actor info.
func (s *Service) List(ctx context.Context, userID uuid.UUID, limit, offset int32) (*ListResponse, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	rows, err := s.repo.ListByUser(ctx, userID, limit+1, offset)
	if err != nil {
		return nil, err
	}

	hasMore := len(rows) > int(limit)
	if hasMore {
		rows = rows[:limit]
	}

	// Collect unique actor IDs for batch enrichment.
	actorIDs := make(map[uuid.UUID]struct{})
	for _, r := range rows {
		if r.ActorID.Valid {
			actorIDs[r.ActorID.Bytes] = struct{}{}
		}
	}

	// Fetch actors.
	actorMap := make(map[uuid.UUID]*ActorDTO, len(actorIDs))
	for id := range actorIDs {
		u, err := s.userRepo.GetByID(ctx, id)
		if err != nil {
			s.log.Warn("failed to fetch notification actor", zap.Error(err))
			continue
		}
		if u != nil {
			actorMap[id] = &ActorDTO{
				ID:          u.ID.String(),
				Username:    u.Username,
				DisplayName: u.DisplayName,
				AvatarURL:   u.AvatarURL,
			}
		}
	}

	dtos := make([]DTO, 0, len(rows))
	for _, r := range rows {
		d := rowToDTO(r)
		if r.ActorID.Valid {
			d.Actor = actorMap[r.ActorID.Bytes]
		}
		dtos = append(dtos, d)
	}

	total, err := s.repo.CountUnread(ctx, userID)
	if err != nil {
		s.log.Warn("failed to count unread", zap.Error(err))
	}

	return &ListResponse{
		Notifications: dtos,
		Total:         total,
		HasMore:       hasMore,
	}, nil
}

// UnreadCount returns the number of unread notifications for a user.
func (s *Service) UnreadCount(ctx context.Context, userID uuid.UUID) (int64, error) {
	return s.repo.CountUnread(ctx, userID)
}

// MarkRead marks a single notification as read.
func (s *Service) MarkRead(ctx context.Context, notifID, userID uuid.UUID) error {
	return s.repo.MarkRead(ctx, notifID, userID)
}

// MarkAllRead marks all unread notifications as read.
func (s *Service) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	return s.repo.MarkAllRead(ctx, userID)
}

// Notify creates a notification with dedup check.
// It is safe to call from a fire-and-forget goroutine.
func (s *Service) Notify(ctx context.Context, p CreateParams) {
	userID, err := uuid.Parse(p.UserID)
	if err != nil {
		s.log.Error("notification: invalid user_id", zap.String("user_id", p.UserID), zap.Error(err))
		return
	}
	actorID, err := uuid.Parse(p.ActorID)
	if err != nil {
		s.log.Error("notification: invalid actor_id", zap.String("actor_id", p.ActorID), zap.Error(err))
		return
	}

	// Don't notify yourself.
	if userID == actorID {
		return
	}

	entityID := pgtype.UUID{}
	if p.EntityID != "" {
		parsed, err := uuid.Parse(p.EntityID)
		if err == nil {
			entityID = pgtype.UUID{Bytes: parsed, Valid: true}
		}
	}

	// Dedup: skip if same actor+type+entity notification exists unread within 24h.
	if p.Type == TypeFollow {
		dup, err := s.repo.CheckDuplicate(ctx, dbgen.CheckDuplicateNotificationParams{
			UserID:   userID,
			ActorID:  pgtype.UUID{Bytes: actorID, Valid: true},
			Type:     p.Type,
			EntityID: entityID,
		})
		if err != nil {
			s.log.Warn("notification: dedup check failed", zap.Error(err))
			// Continue anyway — duplicate is better than missing.
		} else if dup {
			return
		}
	}

	entityType := ptrOrNil(p.EntityType)
	body := ptrOrNil(p.Body)
	url := ptrOrNil(p.URL)

	row, err := s.repo.Create(ctx, dbgen.CreateNotificationParams{
		UserID:     userID,
		ActorID:    pgtype.UUID{Bytes: actorID, Valid: true},
		Type:       p.Type,
		EntityType: entityType,
		EntityID:   entityID,
		Title:      p.Title,
		Body:       body,
		Url:        url,
	})
	if err != nil {
		s.log.Error("notification: failed to create", zap.Error(err))
		return
	}

	// Push real-time SSE event to connected clients.
	if s.hub != nil {
		dto := rowToDTO(row)
		// Enrich actor.
		if u, err := s.userRepo.GetByID(ctx, actorID); err == nil && u != nil {
			dto.Actor = &ActorDTO{
				ID:          u.ID.String(),
				Username:    u.Username,
				DisplayName: u.DisplayName,
				AvatarURL:   u.AvatarURL,
			}
		}
		count, cErr := s.repo.CountUnread(ctx, userID)
		if cErr != nil {
			s.log.Warn("notification: failed to count unread for SSE", zap.Error(cErr))
		}
		s.hub.Publish(p.UserID, Event{
			Type:         "new_notification",
			UnreadCount:  count,
			Notification: &dto,
		})
	}
}

func rowToDTO(r dbgen.Notification) DTO {
	d := DTO{
		ID:        r.ID.String(),
		Type:      r.Type,
		Title:     r.Title,
		Body:      r.Body,
		URL:       r.Url,
		IsRead:    r.IsRead,
		CreatedAt: r.CreatedAt,
	}
	if r.ActorID.Valid {
		s := uuid.UUID(r.ActorID.Bytes).String()
		d.ActorID = &s
	}
	if r.EntityType != nil {
		d.EntityType = r.EntityType
	}
	if r.EntityID.Valid {
		s := uuid.UUID(r.EntityID.Bytes).String()
		d.EntityID = &s
	}
	if r.ReadAt.Valid {
		s := r.ReadAt.Time.Format("2006-01-02T15:04:05Z")
		d.ReadAt = &s
	}
	return d
}

func ptrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
