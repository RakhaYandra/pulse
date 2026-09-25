package usecase

import (
	"context"
	"time"

	"github.com/RakhaYandra/pulse/domain"
)

type Summary struct {
	TotalMonitors   int
	Up              int
	Down            int
	ActiveIncidents int
	Uptime24h       float64
}

type CheckView struct {
	Status         domain.CheckStatus
	StatusCode     *int
	ResponseTimeMs *int
	Error          string
	CheckedAt      time.Time
}

type IncidentView struct {
	ID            string
	MonitorID     string
	MonitorName   string
	Status        domain.IncidentStatus
	Reason        string
	StartedAt     time.Time
	ResolvedAt    *time.Time
	FailureCount  int
	RecoveryCount int
}

type DashboardService struct {
	Dash DashboardRepo
}

func (s DashboardService) Summary(ctx context.Context, userID string) (Summary, error) {
	return s.Dash.Summary(ctx, userID)
}

func (s DashboardService) Checks(ctx context.Context, monitorID, userID string, limit int) ([]CheckView, error) {
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	return s.Dash.Checks(ctx, monitorID, userID, limit)
}

func (s DashboardService) Incidents(ctx context.Context, userID, monitorID string) ([]IncidentView, error) {
	return s.Dash.Incidents(ctx, userID, monitorID, 100)
}
