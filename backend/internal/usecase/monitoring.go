package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/RakhaYandra/pulse/internal/domain"
)

type MonitoringService struct {
	Monitors  MonitorRepo
	Checks    CheckRepo
	Incidents IncidentRepo
	Checker   Checker
}

// Outcome carries what delivery needs for metrics + notify without
// re-querying. Transition is nil when state doesn't flip.
type Outcome struct {
	Result     domain.CheckResult
	Transition *domain.Transition
}

// ProcessCheck records one check result, updates monitor status, and evaluates
// incident transitions.
// ErrNotFound from ActiveMonitor means paused/deleted after enqueue → skip
// silently (nil, nil), preserving old worker behavior.
func (s MonitoringService) ProcessCheck(ctx context.Context, monitorID string) (*Outcome, error) {
	mon, err := s.Monitors.ActiveMonitor(ctx, monitorID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	res := s.Checker.Check(mon.URL, mon.TimeoutSeconds)
	out := &Outcome{Result: res}
	if err := s.Checks.Append(ctx, monitorID, res); err != nil {
		return nil, err
	}
	st := domain.MonitorDown
	if res.Success() {
		st = domain.MonitorUp
	}
	if err := s.Monitors.RecordStatus(ctx, monitorID, st); err != nil {
		return nil, err
	}
	statuses, err := s.Checks.RecentStatuses(ctx, monitorID, max(mon.FailureThreshold, mon.RecoveryThreshold)+1)
	if err != nil {
		return nil, err
	}
	fail, ok := domain.EvalStreak(statuses)
	open, hasOpen := s.openIncident(ctx, monitorID)

	if !res.Success() && fail >= mon.FailureThreshold && !hasOpen {
		detail := describe(res)
		reason := fmt.Sprintf("%d consecutive failures (last: %s)", fail, detail)
		if _, err := s.Incidents.Open(ctx, monitorID, reason, fail); err != nil {
			return nil, err
		}
		out.Transition = &domain.Transition{Type: domain.TransitionOpened, MonitorID: monitorID,
			MonitorName: mon.Name, MonitorURL: mon.URL, FailureCount: fail, Detail: detail}
		return out, nil
	}
	if res.Success() && ok >= mon.RecoveryThreshold && hasOpen {
		if err := s.Incidents.Resolve(ctx, open.ID, ok); err != nil {
			return nil, err
		}
		out.Transition = &domain.Transition{Type: domain.TransitionResolved, MonitorID: monitorID,
			MonitorName: mon.Name, MonitorURL: mon.URL, SuccessCount: ok}
		return out, nil
	}
	return out, nil
}

func (s MonitoringService) openIncident(ctx context.Context, monitorID string) (domain.Incident, bool) {
	inc, has, err := s.Incidents.OpenFor(ctx, monitorID)
	if err != nil {
		return domain.Incident{}, false
	}
	return inc, has
}

func describe(res domain.CheckResult) string {
	if res.StatusCode != 0 {
		return fmt.Sprintf("status %d in %dms", res.StatusCode, res.ResponseTimeMs)
	}
	return fmt.Sprintf("%s: %s", res.Status, res.ErrorMessage)
}
