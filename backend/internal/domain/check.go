package domain

import "time"

type CheckStatus string

const (
	CheckUp      CheckStatus = "UP"
	CheckDown    CheckStatus = "DOWN"
	CheckTimeout CheckStatus = "TIMEOUT"
	CheckError   CheckStatus = "ERROR"
)

// CheckResult is produced by the checker gateway (infrastructure) and
// consumed by the monitoring use-case. One result per check — retries
// collapse inside the checker and never surface here (ADR-003).
type CheckResult struct {
	Status         CheckStatus
	StatusCode     int
	ResponseTimeMs int
	ErrorMessage   string
}

func (r CheckResult) Success() bool { return r.Status == CheckUp }

type Check struct {
	ID             int64
	MonitorID      string
	Status         CheckStatus
	StatusCode     *int
	ResponseTimeMs *int
	ErrorMessage   string
	CheckedAt      time.Time
}
