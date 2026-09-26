package usecase

import (
	"context"
	"strings"
	"time"

	"github.com/RakhaYandra/pulse/internal/domain"
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

// MonitorPatch carries optional update fields. Nil = keep existing value
// (true PATCH semantics; omitted fields are never reset to defaults).
type MonitorPatch struct {
	Name              *string
	URL               *string
	IntervalSeconds   *int
	TimeoutSeconds    *int
	FailureThreshold  *int
	RecoveryThreshold *int
}

// normalize applies creation defaults.
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
	mon.NextRunAt = new(time.Time)
	*mon.NextRunAt = domain.StaggerInitialRunAt(mon.ID, time.Duration(in.IntervalSeconds)*time.Second, time.Now())
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

func (s MonitorService) Update(ctx context.Context, id, userID string, p MonitorPatch) (MonitorDTO, error) {
	existing, err := s.Monitors.ByID(ctx, id, userID)
	if err != nil {
		return MonitorDTO{}, err
	}
	if p.Name != nil {
		existing.Name = strings.TrimSpace(*p.Name)
	}
	if p.URL != nil {
		existing.URL = strings.TrimSpace(*p.URL)
	}
	if p.IntervalSeconds != nil {
		if *p.IntervalSeconds != existing.IntervalSeconds {
			t := time.Now().Add(time.Duration(*p.IntervalSeconds) * time.Second)
			existing.NextRunAt = &t
		}
		existing.IntervalSeconds = *p.IntervalSeconds
	}
	if p.TimeoutSeconds != nil {
		existing.TimeoutSeconds = *p.TimeoutSeconds
	}
	if p.FailureThreshold != nil {
		existing.FailureThreshold = *p.FailureThreshold
	}
	if p.RecoveryThreshold != nil {
		existing.RecoveryThreshold = *p.RecoveryThreshold
	}
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

// Scheduled advances the monitor's next run from the scheduled moment
// (not from check completion), keeping cadence drift-free.
func (s MonitorService) Scheduled(ctx context.Context, id string, at time.Time) error {
	return s.Monitors.MarkScheduled(ctx, id, at)
}
