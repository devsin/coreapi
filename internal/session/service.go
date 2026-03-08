package session

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	dbgen "github.com/devsin/coreapi/gen/db"
)

// Service coordinates session operations.
type Service struct {
	repo *Repository
	log  *zap.Logger

	// In-memory dedup: skip DB upsert if session was seen recently.
	mu       sync.Mutex
	seen     map[string]time.Time // key: "userID:sessionID"
	dedupTTL time.Duration
}

func NewService(repo *Repository, log *zap.Logger) *Service {
	return &Service{
		repo:     repo,
		log:      log,
		seen:     make(map[string]time.Time),
		dedupTTL: 5 * time.Minute,
	}
}

// RecordSession upserts a session in the background with in-memory dedup.
// Safe to call from middleware on every request — only hits DB every dedupTTL.
func (s *Service) RecordSession(
	userID uuid.UUID, sessionID string,
	ip, userAgent, browser, os, deviceType string,
) {
	key := userID.String() + ":" + sessionID

	s.mu.Lock()
	if last, ok := s.seen[key]; ok && time.Since(last) < s.dedupTTL {
		s.mu.Unlock()
		return
	}
	s.seen[key] = time.Now()
	s.mu.Unlock()

	// Fire-and-forget DB upsert (use background context so it completes after request).
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := s.repo.Upsert(bgCtx, dbgen.UpsertSessionParams{
			UserID:     userID,
			SessionID:  sessionID,
			IpAddress:  strPtr(ip),
			UserAgent:  strPtr(userAgent),
			Browser:    strPtr(browser),
			Os:         strPtr(os),
			DeviceType: strPtr(deviceType),
		})
		if err != nil {
			s.log.Error("failed to upsert session", zap.Error(err))
		}
	}()
}

// ListSessions returns all sessions for a user, marking the current one.
func (s *Service) ListSessions(ctx context.Context, userID uuid.UUID, currentSessionID string) (*ListResponse, error) {
	rows, err := s.repo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	sessions := make([]DTO, 0, len(rows))
	for _, r := range rows {
		dto := DTO{
			ID:           r.ID.String(),
			SessionID:    r.SessionID,
			IPAddress:    deref(r.IpAddress),
			Browser:      deref(r.Browser),
			OS:           deref(r.Os),
			DeviceType:   deref(r.DeviceType),
			IsCurrent:    r.SessionID == currentSessionID,
			LastActiveAt: formatTime(r.LastActiveAt),
			CreatedAt:    formatTime(r.CreatedAt),
		}
		sessions = append(sessions, dto)
	}

	return &ListResponse{
		Sessions: sessions,
		Total:    len(sessions),
	}, nil
}

// DeleteSession removes a specific session by ID (must belong to the user).
func (s *Service) DeleteSession(ctx context.Context, sessionDBID, userID uuid.UUID) error {
	return s.repo.Delete(ctx, sessionDBID, userID)
}

// DeleteOtherSessions removes all sessions except the current one.
func (s *Service) DeleteOtherSessions(ctx context.Context, userID uuid.UUID, currentSessionID string) error {
	return s.repo.DeleteOthers(ctx, userID, currentSessionID)
}

// CleanupStaleSessions removes sessions not active in the last 30 days.
func (s *Service) CleanupStaleSessions(ctx context.Context) error {
	return s.repo.CleanupStale(ctx)
}

// CleanupDedupCache removes old entries from the in-memory dedup map.
// Should be called periodically (e.g., every 10 minutes).
func (s *Service) CleanupDedupCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for k, t := range s.seen {
		if now.Sub(t) > s.dedupTTL {
			delete(s.seen, k)
		}
	}
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
