package http

import "time"

// Request/response DTOs. JSON shapes are the API contract (must stay
// identical to pre-CA responses — covered by qa/collection.json).

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type monitorRequest struct {
	Name              string `json:"name"`
	URL               string `json:"url"`
	IntervalSeconds   int    `json:"interval_seconds"`
	TimeoutSeconds    int    `json:"timeout_seconds"`
	FailureThreshold  int    `json:"failure_threshold"`
	RecoveryThreshold int    `json:"recovery_threshold"`
}

type monitorResponse struct {
	ID                string  `json:"id"`
	Name              string  `json:"name"`
	URL               string  `json:"url"`
	Method            string  `json:"method"`
	IntervalSeconds   int     `json:"interval_seconds"`
	TimeoutSeconds    int     `json:"timeout_seconds"`
	FailureThreshold  int     `json:"failure_threshold"`
	RecoveryThreshold int     `json:"recovery_threshold"`
	Status            string  `json:"status"`
	IsActive          bool    `json:"is_active"`
	LastCheckedAt     *string `json:"last_checked_at"`
}

type checkResponse struct {
	Status         string `json:"status"`
	StatusCode     *int   `json:"status_code"`
	ResponseTimeMs *int   `json:"response_time_ms"`
	Error          string `json:"error,omitempty"`
	CheckedAt      string `json:"checked_at"`
}

type incidentResponse struct {
	ID            string  `json:"id"`
	MonitorName   string  `json:"monitor_name"`
	Status        string  `json:"status"`
	Reason        string  `json:"reason"`
	StartedAt     string  `json:"started_at"`
	ResolvedAt    *string `json:"resolved_at"`
	FailureCount  int     `json:"failure_count"`
	RecoveryCount int     `json:"recovery_count"`
}

type summaryResponse struct {
	TotalMonitors   int     `json:"total_monitors"`
	Up              int     `json:"up"`
	Down            int     `json:"down"`
	ActiveIncidents int     `json:"active_incidents"`
	Uptime24h       float64 `json:"uptime_24h"`
}

func fmtTime(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

func fmtTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := fmtTime(*t)
	return &s
}
