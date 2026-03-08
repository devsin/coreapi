package notification

import (
	"context"
)

// UserNotifier wraps the notification service to satisfy the users.Notifier interface.
// users.NotifyParams and notification.CreateParams have the same field layout.
type UserNotifier struct {
	svc *Service
}

// NewUserNotifier creates an adapter for the users service.
func NewUserNotifier(svc *Service) *UserNotifier {
	return &UserNotifier{svc: svc}
}

// Notify converts users.NotifyParams fields and delegates to the notification service.
// This method signature must match users.Notifier.
func (n *UserNotifier) Notify(ctx context.Context, p struct {
	UserID     string
	ActorID    string
	Type       string
	EntityType string
	EntityID   string
	Title      string
	Body       string
	URL        string
},
) {
	n.svc.Notify(ctx, CreateParams(p))
}
