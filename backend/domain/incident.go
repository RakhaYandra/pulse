package domain

import "time"

type IncidentStatus string

const (
	IncidentOpen     IncidentStatus = "OPEN"
	IncidentResolved IncidentStatus = "RESOLVED"
)

type Incident struct {
	ID            string
	MonitorID     string
	MonitorName   string
	Status        IncidentStatus
	Reason        string
	StartedAt     time.Time
	ResolvedAt    *time.Time
	FailureCount  int
	RecoveryCount int
}

type TransitionType string

const (
	TransitionOpened   TransitionType = "OPENED"
	TransitionResolved TransitionType = "RESOLVED"
)

// Transition is returned by the monitoring use-case when a check flips
// incident state. Delivery decides how to notify; formatting lives in
// the notifier adapter, not here.
type Transition struct {
	Type         TransitionType
	MonitorID    string
	MonitorName  string
	MonitorURL   string
	FailureCount int
	SuccessCount int
	Detail       string
}

// EvalStreak counts the trailing consecutive run in statuses (newest first).
// Pure function — unit-testable without any database.
func EvalStreak(statuses []CheckStatus) (fail, ok int) {
	for _, s := range statuses {
		if s == CheckUp {
			if fail > 0 {
				break
			}
			ok++
		} else {
			if ok > 0 {
				break
			}
			fail++
		}
	}
	return fail, ok
}
