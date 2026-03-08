package insights

// OverviewDTO is the top-level insights summary for a user.
type OverviewDTO struct {
	Period           string              `json:"period"` // "7d", "30d", "90d"
	ProfileViews     *MetricDTO          `json:"profile_views"`
	Followers        *MetricDTO          `json:"followers"`
	TimeSeries       *TimeSeriesDTO      `json:"time_series,omitempty"`
	Browsers         []*BreakdownItemDTO `json:"browsers,omitempty"`
	OperatingSystems []*BreakdownItemDTO `json:"operating_systems,omitempty"`
	Devices          []*BreakdownItemDTO `json:"devices,omitempty"`
	Referrers        []*BreakdownItemDTO `json:"referrers,omitempty"`
}

// MetricDTO represents a single metric with current + previous period for comparison.
type MetricDTO struct {
	Current  int64   `json:"current"`
	Previous int64   `json:"previous"`
	Change   float64 `json:"change"` // percentage change
}

// TimeSeriesDTO contains daily data points for charting.
type TimeSeriesDTO struct {
	Dates        []string `json:"dates"`
	ProfileViews []int64  `json:"profile_views"`
}

// EventDTO represents a single insights event (profile view).
type EventDTO struct {
	ID          string   `json:"id"`
	Type        string   `json:"type"` // "profile_view"
	Timestamp   string   `json:"timestamp"`
	IP          *string  `json:"ip,omitempty"`
	Referrer    *string  `json:"referrer,omitempty"`
	Browser     *string  `json:"browser,omitempty"`
	OS          *string  `json:"os,omitempty"`
	DeviceType  *string  `json:"device_type,omitempty"`
	Country     *string  `json:"country,omitempty"`
	CountryCode *string  `json:"country_code,omitempty"`
	City        *string  `json:"city,omitempty"`
	Region      *string  `json:"region,omitempty"`
	Latitude    *float64 `json:"latitude,omitempty"`
	Longitude   *float64 `json:"longitude,omitempty"`
}

// EventsResponse is the paginated response for the events list.
type EventsResponse struct {
	Events []*EventDTO `json:"events"`
	Total  int64       `json:"total"`
	Limit  int32       `json:"limit"`
	Offset int32       `json:"offset"`
}

// BreakdownItemDTO represents a single entry in a browser/OS/device breakdown.
type BreakdownItemDTO struct {
	Name       string  `json:"name"`
	Count      int64   `json:"count"`
	Percentage float64 `json:"percentage"`
}

// GeoPointDTO represents an aggregated location for the map.
type GeoPointDTO struct {
	Country     string  `json:"country"`
	CountryCode string  `json:"country_code"`
	City        string  `json:"city"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	EventCount  int64   `json:"event_count"`
}

// GeoResponse is the response for the map visualization.
type GeoResponse struct {
	Points     []*GeoPointDTO `json:"points"`
	TotalViews int64          `json:"total_views"`
}
