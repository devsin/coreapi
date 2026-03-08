package insights

import (
	"net"
	"net/http"
	"net/netip"
	"strings"

	useragent "github.com/medama-io/go-useragent"
	geoip2 "github.com/oschwald/geoip2-golang/v2"
)

// GeoResult holds the result of a GeoIP lookup.
type GeoResult struct {
	Country     string  `json:"country"`
	CountryCode string  `json:"country_code"`
	Region      string  `json:"region_name"`
	City        string  `json:"city"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
}

// UAResult holds the parsed user-agent information.
type UAResult struct {
	Browser    string
	OS         string
	DeviceType string
}

// GeoIPResolver resolves IP addresses to geographic locations using a local
// MaxMind GeoLite2-City MMDB database (memory-mapped, no network calls).
// The reader is goroutine-safe.
type GeoIPResolver struct {
	db       *geoip2.Reader
	uaParser *useragent.Parser
}

// NewGeoIPResolver opens the MMDB file at dbPath and creates the resolver.
// The caller must call Close() when done (typically via defer).
func NewGeoIPResolver(dbPath string) (*GeoIPResolver, error) {
	db, err := geoip2.Open(dbPath)
	if err != nil {
		return nil, err
	}
	return &GeoIPResolver{
		db:       db,
		uaParser: useragent.NewParser(),
	}, nil
}

// Close releases the MMDB reader resources.
func (r *GeoIPResolver) Close() {
	if r.db != nil {
		r.db.Close()
	}
}

// Resolve looks up geo data for an IP address using the local MMDB database.
// Returns nil if the IP is invalid, private, or not found in the database.
func (r *GeoIPResolver) Resolve(ip string) *GeoResult {
	ip = cleanIP(ip)
	if ip == "" || isPrivateIP(ip) {
		return nil
	}

	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return nil
	}

	record, err := r.db.City(addr)
	if err != nil || !record.HasData() {
		return nil
	}

	result := &GeoResult{
		Country:     record.Country.Names.English,
		CountryCode: record.Country.ISOCode,
		Region:      regionName(record),
		City:        record.City.Names.English,
	}

	if record.Location.HasCoordinates() {
		result.Lat = *record.Location.Latitude
		result.Lon = *record.Location.Longitude
	}

	return result
}

// ParseUserAgent extracts browser, OS, and device type from a User-Agent string
// using the medama-io/go-useragent trie-based parser.
func (r *GeoIPResolver) ParseUserAgent(ua string) UAResult {
	agent := r.uaParser.Parse(ua)
	return UAResult{
		Browser:    string(agent.Browser()),
		OS:         string(agent.OS()),
		DeviceType: string(agent.Device()),
	}
}

// regionName extracts the first subdivision name (state/region) from a City record.
func regionName(record *geoip2.City) string {
	if len(record.Subdivisions) > 0 {
		return record.Subdivisions[0].Names.English
	}
	return ""
}

// cleanIP strips port from IP:port and trims whitespace.
func cleanIP(ip string) string {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return ""
	}
	if host, _, err := net.SplitHostPort(ip); err == nil {
		return host
	}
	return ip
}

// isPrivateIP checks if an IP is private/local.
func isPrivateIP(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return true
	}
	return parsed.IsLoopback() || parsed.IsPrivate() || parsed.IsUnspecified() || parsed.IsLinkLocalUnicast()
}

// ExtractIP extracts the client IP from an HTTP request.
func ExtractIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}
	if realIP := r.Header.Get("X-Real-Ip"); realIP != "" {
		return strings.TrimSpace(realIP)
	}
	return cleanIP(r.RemoteAddr)
}
