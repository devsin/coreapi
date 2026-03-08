package insights

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	dbgen "github.com/devsin/coreapi/gen/db"
)

// Service coordinates insights operations.
type Service struct {
	repo      *Repository
	dailyRepo *DailyInsightsRepo
	log       *zap.Logger
	geo       *GeoIPResolver
}

// NewService creates a new insights service.
func NewService(repo *Repository, dailyRepo *DailyInsightsRepo, geo *GeoIPResolver, log *zap.Logger) *Service {
	return &Service{repo: repo, dailyRepo: dailyRepo, log: log, geo: geo}
}

// periodDuration maps period strings to time.Duration.
func periodDuration(period string) time.Duration {
	switch period {
	case "7d":
		return 7 * 24 * time.Hour
	case "30d":
		return 30 * 24 * time.Hour
	case "90d":
		return 90 * 24 * time.Hour
	default:
		return 30 * 24 * time.Hour
	}
}

// GetOverview builds the full insights overview for a user.
func (s *Service) GetOverview(ctx context.Context, userID uuid.UUID, period string) (*OverviewDTO, error) {
	dur := periodDuration(period)
	now := time.Now().UTC()
	currentStart := now.Add(-dur)
	previousStart := currentStart.Add(-dur)

	profileViews, err := s.buildMetric(ctx, userID, currentStart, now, previousStart, currentStart, s.repo.CountProfileViewsBetween)
	if err != nil {
		s.log.Error("failed to build profile views metric", zap.Error(err))
		profileViews = &MetricDTO{}
	}

	followers, err := s.buildMetric(ctx, userID, currentStart, now, previousStart, currentStart, s.repo.GetNewFollowersBetween)
	if err != nil {
		s.log.Error("failed to build followers metric", zap.Error(err))
		followers = &MetricDTO{}
	}

	timeSeries, err := s.buildTimeSeries(ctx, userID, currentStart, now)
	if err != nil {
		s.log.Error("failed to build time series", zap.Error(err))
	}

	browsers, err := s.buildBreakdown(ctx, userID, currentStart, s.repo.GetBrowserBreakdown)
	if err != nil {
		s.log.Error("failed to get browser breakdown", zap.Error(err))
	}

	operatingSystems, err := s.buildBreakdown(ctx, userID, currentStart, s.repo.GetOSBreakdown)
	if err != nil {
		s.log.Error("failed to get OS breakdown", zap.Error(err))
	}

	devices, err := s.buildBreakdown(ctx, userID, currentStart, s.repo.GetDeviceBreakdown)
	if err != nil {
		s.log.Error("failed to get device breakdown", zap.Error(err))
	}

	referrers, err := s.buildBreakdown(ctx, userID, currentStart, s.repo.GetReferrerBreakdown)
	if err != nil {
		s.log.Error("failed to get referrer breakdown", zap.Error(err))
	}

	return &OverviewDTO{
		Period:           period,
		ProfileViews:     profileViews,
		Followers:        followers,
		TimeSeries:       timeSeries,
		Browsers:         browsers,
		OperatingSystems: operatingSystems,
		Devices:          devices,
		Referrers:        referrers,
	}, nil
}

// countBetweenFunc is a function type for counting metrics between two timestamps.
type countBetweenFunc func(ctx context.Context, userID uuid.UUID, from, to time.Time) (int64, error)

// buildMetric calculates current and previous period counts, then computes the percentage change.
func (s *Service) buildMetric(ctx context.Context, userID uuid.UUID, curFrom, curTo, prevFrom, prevTo time.Time, countFn countBetweenFunc) (*MetricDTO, error) {
	current, err := countFn(ctx, userID, curFrom, curTo)
	if err != nil {
		return nil, fmt.Errorf("counting current period: %w", err)
	}

	previous, err := countFn(ctx, userID, prevFrom, prevTo)
	if err != nil {
		return nil, fmt.Errorf("counting previous period: %w", err)
	}

	change := calculateChange(current, previous)

	return &MetricDTO{
		Current:  current,
		Previous: previous,
		Change:   change,
	}, nil
}

// calculateChange computes percentage change between two values.
func calculateChange(current, previous int64) float64 {
	if previous == 0 {
		if current > 0 {
			return 100.0
		}
		return 0.0
	}
	return float64(current-previous) / float64(previous) * 100.0
}

// buildTimeSeries generates daily data points for charts.
func (s *Service) buildTimeSeries(ctx context.Context, userID uuid.UUID, from, to time.Time) (*TimeSeriesDTO, error) {
	rows, err := s.dailyRepo.GetDailyTimeSeries(ctx, userID, from, to)
	if err != nil {
		s.log.Error("getting daily time series from daily_insights", zap.Error(err))
		return s.buildTimeSeriesFromRawEvents(ctx, userID, from, to)
	}

	days := int(to.Sub(from).Hours()/24) + 1
	dates := make([]string, days)
	profileViews := make([]int64, days)

	for i := 0; i < days; i++ {
		d := from.AddDate(0, 0, i)
		dates[i] = d.Format("2006-01-02")
	}

	dataMap := make(map[string]DailyInsightsRow, len(rows))
	for _, r := range rows {
		key := r.Date.Format("2006-01-02")
		dataMap[key] = r
	}

	for i, d := range dates {
		if row, ok := dataMap[d]; ok {
			profileViews[i] = int64(row.ProfileViews)
		}
	}

	return &TimeSeriesDTO{
		Dates:        dates,
		ProfileViews: profileViews,
	}, nil
}

// buildTimeSeriesFromRawEvents is the legacy fallback.
func (s *Service) buildTimeSeriesFromRawEvents(ctx context.Context, userID uuid.UUID, from, to time.Time) (*TimeSeriesDTO, error) {
	views, err := s.repo.GetDailyProfileViews(ctx, userID, from, to)
	if err != nil {
		s.log.Error("getting daily profile views", zap.Error(err))
	}

	days := int(to.Sub(from).Hours()/24) + 1
	dates := make([]string, days)
	profileViews := make([]int64, days)

	for i := 0; i < days; i++ {
		d := from.AddDate(0, 0, i)
		dates[i] = d.Format("2006-01-02")
	}

	viewMap := make(map[string]int64)
	for _, v := range views {
		if v.Day.Valid {
			key := v.Day.Time.Format("2006-01-02")
			viewMap[key] = v.Views
		}
	}

	for i, d := range dates {
		profileViews[i] = viewMap[d]
	}

	return &TimeSeriesDTO{
		Dates:        dates,
		ProfileViews: profileViews,
	}, nil
}

// breakdownFunc fetches a breakdown from the repository.
type breakdownFunc func(ctx context.Context, userID uuid.UUID, since time.Time) ([]BreakdownItem, error)

// buildBreakdown fetches raw counts and converts them to DTOs with percentages.
func (s *Service) buildBreakdown(ctx context.Context, userID uuid.UUID, since time.Time, fn breakdownFunc) ([]*BreakdownItemDTO, error) {
	items, err := fn(ctx, userID, since)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}

	var total int64
	for _, item := range items {
		total += item.Count
	}

	dtos := make([]*BreakdownItemDTO, len(items))
	for i, item := range items {
		pct := 0.0
		if total > 0 {
			pct = float64(item.Count) / float64(total) * 100.0
		}
		dtos[i] = &BreakdownItemDTO{
			Name:       item.Name,
			Count:      item.Count,
			Percentage: math.Round(pct*10) / 10, // round to 1 decimal
		}
	}
	return dtos, nil
}

// RecordProfileView records a profile view event with geo/UA enrichment (fire-and-forget friendly).
// Skips recording if the same IP viewed the same profile within the dedup window (30 min).
func (s *Service) RecordProfileView(ctx context.Context, profileUserID uuid.UUID, ip, userAgent, referrer string) {
	if ip != "" {
		isDup, err := s.repo.CheckDuplicateProfileView(ctx, profileUserID, ip)
		if err != nil {
			s.log.Error("failed to check duplicate profile view", zap.Error(err))
		} else if isDup {
			s.log.Debug("skipping duplicate profile view",
				zap.String("profile_user_id", profileUserID.String()),
				zap.String("ip", ip))
			return
		}
	}

	params := s.buildRecordParams(ip, userAgent, referrer)

	if err := s.repo.RecordProfileView(ctx, dbgen.RecordProfileViewParams{
		ProfileUserID: profileUserID,
		ViewerIp:      strPtr(ip),
		Referrer:      strPtr(referrer),
		UserAgent:     strPtr(userAgent),
		Browser:       params.browser,
		Os:            params.os,
		DeviceType:    params.deviceType,
		Country:       params.country,
		CountryCode:   params.countryCode,
		City:          params.city,
		Region:        params.region,
		Latitude:      params.latitude,
		Longitude:     params.longitude,
	}); err != nil {
		s.log.Error("failed to record profile view", zap.String("profile_user_id", profileUserID.String()), zap.Error(err))
	}

	if err := s.dailyRepo.IncrementDailyProfileViews(ctx, profileUserID); err != nil {
		s.log.Error("failed to increment daily profile views", zap.Error(err))
	}
}

type enrichedParams struct {
	browser     *string
	os          *string
	deviceType  *string
	country     *string
	countryCode *string
	city        *string
	region      *string
	latitude    *float64
	longitude   *float64
}

func (s *Service) buildRecordParams(ip, userAgent, _ string) enrichedParams {
	var p enrichedParams

	if userAgent != "" {
		ua := s.geo.ParseUserAgent(userAgent)
		if ua.Browser != "" {
			p.browser = &ua.Browser
		}
		if ua.OS != "" {
			p.os = &ua.OS
		}
		if ua.DeviceType != "" {
			p.deviceType = &ua.DeviceType
		}
	}

	if ip != "" {
		if geo := s.geo.Resolve(ip); geo != nil {
			p.country = &geo.Country
			p.countryCode = &geo.CountryCode
			p.city = &geo.City
			p.region = &geo.Region
			p.latitude = &geo.Lat
			p.longitude = &geo.Lon
		}
	}

	return p
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// GetEvents returns a paginated list of profile view events.
func (s *Service) GetEvents(ctx context.Context, userID uuid.UUID, limit, offset int32) (*EventsResponse, error) {
	total, err := s.repo.CountProfileViewEvents(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("counting view events: %w", err)
	}

	rows, err := s.repo.GetRecentProfileViewEvents(ctx, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("getting view events: %w", err)
	}

	events := make([]*EventDTO, len(rows))
	for i, row := range rows {
		events[i] = &EventDTO{
			ID:          row.ID.String(),
			Type:        "profile_view",
			Timestamp:   row.ViewedAt.Format(time.RFC3339),
			IP:          maskIP(row.ViewerIp),
			Referrer:    row.Referrer,
			Browser:     row.Browser,
			OS:          row.Os,
			DeviceType:  row.DeviceType,
			Country:     row.Country,
			CountryCode: row.CountryCode,
			City:        row.City,
			Region:      row.Region,
			Latitude:    row.Latitude,
			Longitude:   row.Longitude,
		}
	}

	return &EventsResponse{Events: events, Total: total, Limit: limit, Offset: offset}, nil
}

// maskIP masks the last octet of an IPv4 address for privacy.
func maskIP(ip *string) *string {
	if ip == nil || *ip == "" {
		return nil
	}
	parts := strings.Split(*ip, ".")
	if len(parts) == 4 {
		masked := parts[0] + "." + parts[1] + "." + parts[2] + ".***"
		return &masked
	}
	if strings.Contains(*ip, ":") {
		idx := strings.LastIndex(*ip, ":")
		masked := (*ip)[:idx] + ":****"
		return &masked
	}
	return ip
}

// GetGeoData returns aggregated geo data for map visualization.
func (s *Service) GetGeoData(ctx context.Context, userID uuid.UUID) (*GeoResponse, error) {
	viewGeo, err := s.repo.GetProfileViewGeoAggregation(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("getting view geo: %w", err)
	}

	points := make([]*GeoPointDTO, 0, len(viewGeo))
	var totalViews int64

	for _, row := range viewGeo {
		totalViews += row.EventCount
		points = append(points, &GeoPointDTO{
			Country:     deref(row.Country),
			CountryCode: deref(row.CountryCode),
			City:        deref(row.City),
			Latitude:    row.Latitude,
			Longitude:   row.Longitude,
			EventCount:  row.EventCount,
		})
	}

	sort.Slice(points, func(i, j int) bool {
		return points[i].EventCount > points[j].EventCount
	})

	return &GeoResponse{
		Points:     points,
		TotalViews: totalViews,
	}, nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// BackfillDailyInsights aggregates raw event data into the daily_insights table.
func (s *Service) BackfillDailyInsights(ctx context.Context, from, to time.Time) error {
	s.log.Info("starting daily insights backfill",
		zap.Time("from", from),
		zap.Time("to", to),
	)

	if err := s.dailyRepo.BackfillAll(ctx, from, to); err != nil {
		return fmt.Errorf("backfill daily insights: %w", err)
	}

	s.log.Info("daily insights backfill completed")
	return nil
}
