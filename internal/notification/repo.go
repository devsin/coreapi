package notification

import (
	"context"

	dbgen "github.com/devsin/coreapi/gen/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository provides data access for notifications.
type Repository struct {
	q *dbgen.Queries
}

// NewRepository creates a new notification repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{q: dbgen.New(pool)}
}

// Create inserts a new notification row.
func (r *Repository) Create(ctx context.Context, p dbgen.CreateNotificationParams) (dbgen.Notification, error) {
	return r.q.CreateNotification(ctx, p)
}

// ListByUser returns paginated notifications for a user.
func (r *Repository) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]dbgen.Notification, error) {
	return r.q.ListNotificationsByUser(ctx, dbgen.ListNotificationsByUserParams{
		UserID: userID,
		Limit:  limit,
		Offset: offset,
	})
}

// CountUnread returns the number of unread notifications for a user.
func (r *Repository) CountUnread(ctx context.Context, userID uuid.UUID) (int64, error) {
	return r.q.CountUnreadNotifications(ctx, userID)
}

// MarkRead marks a single notification as read.
func (r *Repository) MarkRead(ctx context.Context, id, userID uuid.UUID) error {
	return r.q.MarkNotificationRead(ctx, dbgen.MarkNotificationReadParams{
		ID:     id,
		UserID: userID,
	})
}

// MarkAllRead marks all unread notifications as read for a user.
func (r *Repository) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	return r.q.MarkAllNotificationsRead(ctx, userID)
}

// CheckDuplicate checks if an unread notification exists for the same actor+type+entity within 24h.
func (r *Repository) CheckDuplicate(ctx context.Context, p dbgen.CheckDuplicateNotificationParams) (bool, error) {
	return r.q.CheckDuplicateNotification(ctx, p)
}

// Delete removes a notification.
func (r *Repository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	return r.q.DeleteNotification(ctx, dbgen.DeleteNotificationParams{
		ID:     id,
		UserID: userID,
	})
}
