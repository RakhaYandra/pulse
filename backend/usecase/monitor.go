package usecase

import (
	"context"
	"strings"
	"time"

	"github.com/RakhaYandra/pulse/domain"
	"github.com/google/uuid"
)

type MonitorDTO struct {
	ID                string
	Name              string
	URL               string
	Method            string
	IntervalSeconds   int
	TimeoutSeconds    int
	FailureThreshold  int
	RecoveryThreshold int
	Status            domain.MonitorStatus
	IsActive          bool
	LastCheckedAt     *time.Time
}

type MonitorInput struct {
	Name              string
	URL               string
	IntervalSeconds   int
	TimeoutSeconds    int
	FailureThreshold  int
	RecoveryThreshold int
}

// normalize applies creation defaults. Known quirk (preserved): update is
// full-replace, so omitted thresholds reset to defaults.
func (in *MonitorInput) normalize() {
	if in.IntervalSeconds == 0 {
		in.IntervalSeconds = 300
	}
	if in.TimeoutSeconds == 0 {
		in.TimeoutSeconds = 5
	}
	if in.FailureThreshold == 0 {
		in.FailureThreshold = 3
	}
	if in.RecoveryThreshold == 0 {
		in.RecoveryThreshold = 2
	}
	in.Name = strings.TrimSpace(in.Name)
	in.URL = strings.TrimSpace(in.URL)
}

type MonitorService struct {
	Monitors MonitorRepo
}

func toMonitorDTO(m domain.Monitor) MonitorDTO {
	return MonitorDTO{
		ID: m.ID, Name: m.Name, URL: m.URL, Method: m.Method,
		IntervalSeconds: m.IntervalSeconds, TimeoutSeconds: m.TimeoutSeconds,
		FailureThreshold: m.FailureThreshold, RecoveryThreshold: m.RecoveryThreshold,
		Status: m.Status, IsActive: m.IsActive, LastCheckedAt: m.LastCheckedAt,
	}
}

func (s MonitorService) Create(ctx context.Context, userID string, in MonitorInput) (MonitorDTO, error) {
	in.normalize()
	mon := domain.Monitor{
		ID: uuid.NewString(), UserID: userID,
		Name: in.Name, URL: in.URL, Method: "GET",
		IntervalSeconds: in.IntervalSeconds, TimeoutSeconds: in.TimeoutSeconds,
		FailureThreshold: in.FailureThreshold, RecoveryThreshold: in.RecoveryThreshold,
		Status: domain.MonitorUnknown, IsActive: true,
	}
	if err := mon.Validate(); err != nil {
		return MonitorDTO{}, err
	}
	if err := s.Monitors.Create(ctx, mon); err != nil {
		return MonitorDTO{}, err
	}
	saved, err := s.Monitors.ByID(ctx, mon.ID, userID)
	if err != nil {
		return MonitorDTO{}, err
	}
	return toMonitorDTO(saved), nil
}

func (s MonitorService) List(ctx context.Context, userID string) ([]MonitorDTO, error) {
	ms, err := s.Monitors.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]MonitorDTO, 0, len(ms))
	for _, m := range ms {
		out = append(out, toMonitorDTO(m))
	}
	return out, nil
}

func (s MonitorService) Get(ctx context.Context, id, userID string) (MonitorDTO, error) {
	m, err := s.Monitors.ByID(ctx, id, userID)
	if err != nil {
		return MonitorDTO{}, err
	}
	return toMonitorDTO(m), nil
}

func (s MonitorService) Update(ctx context.Context, id, userID string, in MonitorInput) (MonitorDTO, error) {
	in.normalize()
	existing, err := s.Monitors.ByID(ctx, id, userID)
	if err != nil {
		return MonitorDTO{}, err
	}
	existing.Name = in.Name
	existing.URL = in.URL
	existing.IntervalSeconds = in.IntervalSeconds
	existing.TimeoutSeconds = in.TimeoutSeconds
	existing.FailureThreshold = in.FailureThreshold
	existing.RecoveryThreshold = in.RecoveryThreshold
	if err := existing.Validate(); err != nil {
		return MonitorDTO{}, err
	}
	if err := s.Monitors.Update(ctx, existing); err != nil {
		return MonitorDTO{}, err
	}
	updated, err := s.Monitors.ByID(ctx, id, userID)
	if err != nil {
		return MonitorDTO{}, err
	}
	return toMonitorDTO(updated), nil
}

func (s MonitorService) Delete(ctx context.Context, id, userID string) error {
	return s.Monitors.Delete(ctx, id, userID)
}

func (s MonitorService) SetActive(ctx context.Context, id, userID string, active bool) (MonitorDTO, error) {
	m, err := s.Monitors.SetActive(ctx, id, userID, active)
	if err != nil {
		return MonitorDTO{}, err
	}
	return toMonitorDTO(m), nil
}

// Due lists monitors whose schedule elapsed (scheduler runner only).
func (s MonitorService) Due(ctx context.Context) ([]MonitorDTO, error) {
	ms, err := s.Monitors.FindDue(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]MonitorDTO, 0, len(ms))
	for _, m := range ms {
		out = append(out, toMonitorDTO(m))
	}
	return out, nil
}
