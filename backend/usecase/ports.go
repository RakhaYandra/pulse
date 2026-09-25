package usecase

import (
	"context"
	"time"

	"github.com/RakhaYandra/pulse/domain"
)

// Ports: all interfaces owned by the use-case layer (dependency inversion).
// Infrastructure implements them; delivery only sees use-case services.

type UserRepo interface {
	Create(ctx context.Context, u domain.User) error // ErrConflict on dup email
	ByEmail(ctx context.Context, email string) (domain.User, error)
	ByID(ctx context.Context, id string) (domain.User, error)
}

type MonitorRepo interface {
	Create(ctx context.Context, m domain.Monitor) error
	ByID(ctx context.Context, id, userID string) (domain.Monitor, error) // ErrNotFound if not owned
	ListByUser(ctx context.Context, userID string) ([]domain.Monitor, error)
	Update(ctx context.Context, m domain.Monitor) error // ErrNotFound if not owned
	Delete(ctx context.Context, id, userID string) error
	SetActive(ctx context.Context, id, userID string, active bool) (domain.Monitor, error)
	// FindDue returns active monitors whose schedule elapsed (scheduler only).
	FindDue(ctx context.Context) ([]domain.Monitor, error)
	// ActiveMonitor returns url+thresholds iff monitor exists and is active.
	ActiveMonitor(ctx context.Context, id string) (domain.Monitor, error) // ErrNotFound skips job
	RecordStatus(ctx context.Context, id string, st domain.MonitorStatus) error
	// MarkScheduled advances next_run_at from the scheduled moment.
	MarkScheduled(ctx context.Context, id string, next time.Time) error
}

type CheckRepo interface {
	Append(ctx context.Context, monitorID string, r domain.CheckResult) error
	// RecentStatuses returns newest-first statuses, up to limit.
	RecentStatuses(ctx context.Context, monitorID string, limit int) ([]domain.CheckStatus, error)
}

type IncidentRepo interface {
	Open(ctx context.Context, monitorID, reason string, failCount int) (domain.Incident, error)
	OpenFor(ctx context.Context, monitorID string) (domain.Incident, bool, error)
	Resolve(ctx context.Context, id string, recoveryCount int) error
}

type DashboardRepo interface {
	Summary(ctx context.Context, userID string) (Summary, error)
	Checks(ctx context.Context, monitorID, userID string, limit int) ([]CheckView, error)
	Incidents(ctx context.Context, userID, monitorID string, limit int) ([]IncidentView, error)
	Reliability(ctx context.Context, userID string, days int) ([]ReliabilityRow, error)
}

type Hasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) error
}

type TokenIssuer interface {
	Issue(userID string) (string, error)
	// Parse validates a token and returns the subject (delivery middleware).
	Parse(token string) (string, error)
}

type Checker interface {
	Check(url string, timeoutSec int) domain.CheckResult
}

type Notifier interface {
	Notify(t domain.Transition)
}

type JobQueue interface {
	// Enqueue queues one job unless an unexpired claim exists (dedup).
	// Returns true when the job was actually queued.
	Enqueue(ctx context.Context, monitorID string, ttl time.Duration) (bool, error)
	// Release drops the dedup claim after a job finishes. TTL remains as
	// crash safety net — a crashed job is re-queued once the claim expires.
	Release(ctx context.Context, monitorID string) error
	// Dequeue blocks up to timeout; returns "" with nil error on timeout.
	Dequeue(ctx context.Context, timeout time.Duration) (string, error)
	// Depth returns pending jobs (Redis LLEN). Used for metrics only.
	Depth(ctx context.Context) (int64, error)
}
