package insights

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	dbgen "github.com/devsin/coreapi/gen/db"
)

// DailyInsightsRepo handles daily_insights operations.
type DailyInsightsRepo struct {
	pool *pgxpool.Pool
	q    *dbgen.Queries
}

// NewDailyInsightsRepo creates a new daily insights repository.
func NewDailyInsightsRepo(pool *pgxpool.Pool) *DailyInsightsRepo {
	return &DailyInsightsRepo{pool: pool, q: dbgen.New(pool)}
}

// DailyInsightsRow represents a single row from the daily_insights table.
type DailyInsightsRow struct {
	Date         time.Time
	ProfileViews int32
	NewFollowers int32
}

// IncrementDailyProfileViews atomically increments today's profile_views for a user.
func (r *DailyInsightsRepo) IncrementDailyProfileViews(ctx context.Context, userID uuid.UUID) error {
	return r.q.IncrementDailyProfileViews(ctx, userID)
}

// GetDailyTimeSeries retrieves pre-aggregated daily stats for a user within a date range.
func (r *DailyInsightsRepo) GetDailyTimeSeries(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]DailyInsightsRow, error) {
	rows, err := r.q.GetDailyInsights(ctx, dbgen.GetDailyInsightsParams{
		UserID: userID,
		Date:   pgtype.Date{Time: from, Valid: true},
		Date_2: pgtype.Date{Time: to, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	result := make([]DailyInsightsRow, len(rows))
	for i, row := range rows {
		result[i] = DailyInsightsRow{
			Date:         row.Date.Time,
			ProfileViews: row.ProfileViews,
			NewFollowers: row.NewFollowers,
		}
	}
	return result, nil
}

// BackfillAll runs all backfill queries for the given date range in a single transaction.
func (r *DailyInsightsRepo) BackfillAll(ctx context.Context, from, to time.Time) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback after commit is a no-op

	qtx := r.q.WithTx(tx)

	if err := qtx.BackfillDailyProfileViews(ctx, dbgen.BackfillDailyProfileViewsParams{
		ViewedAt:   from,
		ViewedAt_2: to,
	}); err != nil {
		return err
	}

	if err := qtx.BackfillDailyNewFollowers(ctx, dbgen.BackfillDailyNewFollowersParams{
		CreatedAt:   from,
		CreatedAt_2: to,
	}); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
