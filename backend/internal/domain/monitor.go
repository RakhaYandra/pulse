package domain

import (
	"hash/fnv"
	"net/url"
	"strings"
	"time"
)

type MonitorStatus string

const (
	MonitorUnknown MonitorStatus = "UNKNOWN"
	MonitorUp      MonitorStatus = "UP"
	MonitorDown    MonitorStatus = "DOWN"
	MonitorPaused  MonitorStatus = "PAUSED"
)

type Monitor struct {
	ID                string
	UserID            string
	Name              string
	URL               string
	Method            string
	IntervalSeconds   int
	TimeoutSeconds    int
	FailureThreshold  int
	RecoveryThreshold int
	Status            MonitorStatus
	IsActive          bool
	LastCheckedAt     *time.Time
	// NextRunAt is internal scheduling state (not exposed via API).
	// Due = NextRunAt elapsed, so cadence doesn't drift with drain time.
	NextRunAt *time.Time
	CreatedAt time.Time
}

// Validate enforces creation/update business rules. No framework, no SQL.
func (m Monitor) Validate() error {
	if strings.TrimSpace(m.Name) == "" {
		return &FieldError{Field: "name", Message: "name required"}
	}
	u, err := url.ParseRequestURI(strings.TrimSpace(m.URL))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return &FieldError{Field: "url", Message: "url must be valid http(s)"}
	}
	host := strings.ToLower(u.Hostname())
	if BlockedHost(host) {
		return &FieldError{Field: "url", Message: "url host not allowed"}
	}
	if m.IntervalSeconds < 60 {
		return &FieldError{Field: "interval_seconds", Message: "interval_seconds min 60"}
	}
	if m.TimeoutSeconds < 1 || m.TimeoutSeconds > 60 {
		return &FieldError{Field: "timeout_seconds", Message: "timeout_seconds 1..60"}
	}
	if m.TimeoutSeconds >= m.IntervalSeconds {
		return &FieldError{Field: "timeout_seconds", Message: "timeout must be < interval"}
	}
	if m.FailureThreshold < 1 || m.RecoveryThreshold < 1 {
		return &FieldError{Field: "failure_threshold", Message: "thresholds min 1"}
	}
	return nil
}

// StaggerInitialRunAt spreads a new monitor's first run across [now,
// now+interval) using a deterministic hash of its ID. Monitors created
// together no longer fire as one synchronized wave; same ID always yields
// the same offset (reproducible, testable, no RNG state).
func StaggerInitialRunAt(id string, interval time.Duration, now time.Time) time.Time {
	if interval <= 0 {
		return now
	}
	h := fnv.New32a()
	h.Write([]byte(id))
	offset := time.Duration(h.Sum32() % uint32(interval.Seconds()))
	return now.Add(offset * time.Second)
}
