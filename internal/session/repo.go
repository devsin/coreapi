package session

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	dbgen "github.com/devsin/coreapi/gen/db"
)

// Repository encapsulates database operations for user sessions.
type Repository struct {
	pool *pgxpool.Pool
	q    *dbgen.Queries
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool, q: dbgen.New(pool)}
}

func (r *Repository) Upsert(ctx context.Context, params dbgen.UpsertSessionParams) error {
	return r.q.UpsertSession(ctx, params)
}

func (r *Repository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]dbgen.UserSession, error) {
	return r.q.ListSessionsByUserID(ctx, userID)
}

func (r *Repository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	return r.q.DeleteSession(ctx, dbgen.DeleteSessionParams{
		ID:     id,
		UserID: userID,
	})
}

func (r *Repository) DeleteOthers(ctx context.Context, userID uuid.UUID, currentSessionID string) error {
	return r.q.DeleteOtherSessions(ctx, dbgen.DeleteOtherSessionsParams{
		UserID:    userID,
		SessionID: currentSessionID,
	})
}

func (r *Repository) DeleteAll(ctx context.Context, userID uuid.UUID) error {
	return r.q.DeleteAllUserSessions(ctx, userID)
}

func (r *Repository) CleanupStale(ctx context.Context) error {
	return r.q.CleanupStaleSessions(ctx)
}
