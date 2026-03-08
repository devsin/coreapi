package insights

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	dbgen "github.com/devsin/coreapi/gen/db"
)

// Repository encapsulates database operations for insights.
type Repository struct {
	pool *pgxpool.Pool
	q    *dbgen.Queries
}

// NewRepository creates a new insights repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool, q: dbgen.New(pool)}
}

// RecordProfileView records a single profile view event with all metadata.
func (r *Repository) RecordProfileView(ctx context.Context, arg dbgen.RecordProfileViewParams) error {
	return r.q.RecordProfileView(ctx, arg)
}

// CheckDuplicateProfileView returns true if the same IP viewed the same profile within the dedup window.
func (r *Repository) CheckDuplicateProfileView(ctx context.Context, profileUserID uuid.UUID, ip string) (bool, error) {
	return r.q.CheckDuplicateProfileView(ctx, dbgen.CheckDuplicateProfileViewParams{
		ProfileUserID: profileUserID,
		ViewerIp:      &ip,
	})
}

// CountProfileViewsBetween counts profile views between two timestamps.
func (r *Repository) CountProfileViewsBetween(ctx context.Context, userID uuid.UUID, from, to time.Time) (int64, error) {
	return r.q.CountProfileViewsBetween(ctx, dbgen.CountProfileViewsBetweenParams{
		ProfileUserID: userID,
		ViewedAt:      from,
		ViewedAt_2:    to,
	})
}

// GetNewFollowersBetween counts new followers between two timestamps.
func (r *Repository) GetNewFollowersBetween(ctx context.Context, userID uuid.UUID, from, to time.Time) (int64, error) {
	return r.q.GetNewFollowersBetween(ctx, dbgen.GetNewFollowersBetweenParams{
		FollowingID: userID,
		CreatedAt:   from,
		CreatedAt_2: to,
	})
}

// GetDailyProfileViews returns daily profile view counts.
func (r *Repository) GetDailyProfileViews(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]dbgen.GetDailyProfileViewsRow, error) {
	return r.q.GetDailyProfileViews(ctx, dbgen.GetDailyProfileViewsParams{
		ProfileUserID: userID,
		ViewedAt:      from,
		ViewedAt_2:    to,
	})
}

// GetRecentProfileViewEvents returns paginated profile view events.
func (r *Repository) GetRecentProfileViewEvents(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]dbgen.ProfileView, error) {
	return r.q.GetRecentProfileViewEvents(ctx, dbgen.GetRecentProfileViewEventsParams{
		ProfileUserID: userID,
		Limit:         limit,
		Offset:        offset,
	})
}

// CountProfileViewEvents counts total profile view events.
func (r *Repository) CountProfileViewEvents(ctx context.Context, userID uuid.UUID) (int64, error) {
	return r.q.CountProfileViewEvents(ctx, userID)
}

// GetProfileViewGeoAggregation returns geo-aggregated profile view data.
func (r *Repository) GetProfileViewGeoAggregation(ctx context.Context, userID uuid.UUID) ([]dbgen.GetProfileViewGeoAggregationRow, error) {
	return r.q.GetProfileViewGeoAggregation(ctx, userID)
}

// BreakdownItem is a single name+count pair used for browser/OS/device aggregation.
type BreakdownItem struct {
	Name  string
	Count int64
}

// GetBrowserBreakdown returns browser counts from profile view events.
func (r *Repository) GetBrowserBreakdown(ctx context.Context, userID uuid.UUID, since time.Time) ([]BreakdownItem, error) {
	rows, err := r.q.GetProfileViewBrowserCounts(ctx, dbgen.GetProfileViewBrowserCountsParams{
		ProfileUserID: userID, ViewedAt: since,
	})
	if err != nil {
		return nil, err
	}

	items := make([]BreakdownItem, 0, len(rows))
	for _, row := range rows {
		if row.Browser != nil {
			items = append(items, BreakdownItem{Name: *row.Browser, Count: row.EventCount})
		}
	}
	return sortedBreakdown(items, 10), nil
}

// GetOSBreakdown returns OS counts from profile view events.
func (r *Repository) GetOSBreakdown(ctx context.Context, userID uuid.UUID, since time.Time) ([]BreakdownItem, error) {
	rows, err := r.q.GetProfileViewOSCounts(ctx, dbgen.GetProfileViewOSCountsParams{
		ProfileUserID: userID, ViewedAt: since,
	})
	if err != nil {
		return nil, err
	}

	items := make([]BreakdownItem, 0, len(rows))
	for _, row := range rows {
		if row.Os != nil {
			items = append(items, BreakdownItem{Name: *row.Os, Count: row.EventCount})
		}
	}
	return sortedBreakdown(items, 10), nil
}

// GetDeviceBreakdown returns device type counts from profile view events.
func (r *Repository) GetDeviceBreakdown(ctx context.Context, userID uuid.UUID, since time.Time) ([]BreakdownItem, error) {
	rows, err := r.q.GetProfileViewDeviceCounts(ctx, dbgen.GetProfileViewDeviceCountsParams{
		ProfileUserID: userID, ViewedAt: since,
	})
	if err != nil {
		return nil, err
	}

	items := make([]BreakdownItem, 0, len(rows))
	for _, row := range rows {
		if row.DeviceType != nil {
			items = append(items, BreakdownItem{Name: *row.DeviceType, Count: row.EventCount})
		}
	}
	return sortedBreakdown(items, 10), nil
}

// GetReferrerBreakdown returns referrer counts from profile view events, extracting domain from URL.
func (r *Repository) GetReferrerBreakdown(ctx context.Context, userID uuid.UUID, since time.Time) ([]BreakdownItem, error) {
	rows, err := r.q.GetProfileViewReferrerCounts(ctx, dbgen.GetProfileViewReferrerCountsParams{
		ProfileUserID: userID, ViewedAt: since,
	})
	if err != nil {
		return nil, err
	}

	merged := make(map[string]int64)
	for _, row := range rows {
		if row.Referrer != nil {
			merged[extractDomain(*row.Referrer)] += row.EventCount
		}
	}

	items := make([]BreakdownItem, 0, len(merged))
	for name, count := range merged {
		items = append(items, BreakdownItem{Name: name, Count: count})
	}
	return sortedBreakdown(items, 10), nil
}

// extractDomain parses a referrer URL and returns just the hostname (e.g. "google.com").
func extractDomain(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "(direct)"
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return raw
	}
	host := strings.TrimPrefix(u.Hostname(), "www.")
	if host == "" {
		return raw
	}
	return host
}

// sortedBreakdown sorts items by count descending and caps at limit.
func sortedBreakdown(items []BreakdownItem, limit int) []BreakdownItem { //nolint:unparam // limit kept as parameter for flexibility
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j].Count > items[i].Count {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
	if len(items) > limit {
		items = items[:limit]
	}
	return items
}
