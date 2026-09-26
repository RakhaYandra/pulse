package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/RakhaYandra/pulse/internal/domain"
)

// In-memory fakes: prove use-case logic without any database.

type fakeMonitorRepo struct {
	m      domain.Monitor
	checks []domain.CheckStatus
}

func (f *fakeMonitorRepo) Create(ctx context.Context, m domain.Monitor) error { f.m = m; return nil }
func (f *fakeMonitorRepo) ByID(ctx context.Context, id, userID string) (domain.Monitor, error) {
	if id != f.m.ID || userID != f.m.UserID {
		return domain.Monitor{}, domain.ErrNotFound
	}
	return f.m, nil
}
func (f *fakeMonitorRepo) ListByUser(ctx context.Context, userID string) ([]domain.Monitor, error) {
	return []domain.Monitor{f.m}, nil
}
func (f *fakeMonitorRepo) Update(ctx context.Context, m domain.Monitor) error  { f.m = m; return nil }
func (f *fakeMonitorRepo) Delete(ctx context.Context, id, userID string) error { return nil }
func (f *fakeMonitorRepo) SetActive(ctx context.Context, id, userID string, a bool) (domain.Monitor, error) {
	f.m.IsActive = a
	return f.m, nil
}
func (f *fakeMonitorRepo) FindDue(ctx context.Context) ([]domain.Monitor, error) {
	return []domain.Monitor{f.m}, nil
}
func (f *fakeMonitorRepo) ActiveMonitor(ctx context.Context, id string) (domain.Monitor, error) {
	if !f.m.IsActive {
		return domain.Monitor{}, domain.ErrNotFound
	}
	return f.m, nil
}
func (f *fakeMonitorRepo) RecordStatus(ctx context.Context, id string, st domain.MonitorStatus) error {
	f.m.Status = st
	return nil
}
func (f *fakeMonitorRepo) MarkScheduled(ctx context.Context, id string, next time.Time) error {
	f.m.NextRunAt = &next
	return nil
}

type fakeCheckRepo struct{ statuses []domain.CheckStatus }

func (f *fakeCheckRepo) Append(ctx context.Context, id string, r domain.CheckResult) error {
	f.statuses = append([]domain.CheckStatus{r.Status}, f.statuses...)
	return nil
}
func (f *fakeCheckRepo) RecentStatuses(ctx context.Context, id string, limit int) ([]domain.CheckStatus, error) {
	if len(f.statuses) > limit {
		return f.statuses[:limit], nil
	}
	return f.statuses, nil
}

type fakeIncidentRepo struct {
	open   *domain.Incident
	opened int
}

func (f *fakeIncidentRepo) Open(ctx context.Context, mid, reason string, fails int) (domain.Incident, error) {
	f.opened++
	inc := domain.Incident{ID: "inc-1", MonitorID: mid, Status: domain.IncidentOpen, Reason: reason, FailureCount: fails}
	f.open = &inc
	return inc, nil
}
func (f *fakeIncidentRepo) OpenFor(ctx context.Context, mid string) (domain.Incident, bool, error) {
	if f.open == nil {
		return domain.Incident{}, false, nil
	}
	return *f.open, true, nil
}
func (f *fakeIncidentRepo) Resolve(ctx context.Context, id string, n int) error {
	f.open.Status = domain.IncidentResolved
	f.open.RecoveryCount = n
	return nil
}

type scriptChecker struct{ results []domain.CheckResult }

func (s *scriptChecker) Check(url string, timeoutSec int) domain.CheckResult {
	r := s.results[0]
	s.results = s.results[1:]
	return r
}

func down(code int) domain.CheckResult {
	return domain.CheckResult{Status: domain.CheckDown, StatusCode: code, ResponseTimeMs: 100, ErrorMessage: "unexpected"}
}
func up() domain.CheckResult {
	return domain.CheckResult{Status: domain.CheckUp, StatusCode: 200, ResponseTimeMs: 50}
}

func testMonitor() domain.Monitor {
	return domain.Monitor{ID: "m1", UserID: "u1", Name: "T", URL: "https://x.com",
		Method: "GET", IntervalSeconds: 60, TimeoutSeconds: 5,
		FailureThreshold: 3, RecoveryThreshold: 2, Status: domain.MonitorUnknown, IsActive: true}
}

func TestProcessCheckOpensIncident(t *testing.T) {
	mr, cr, ir := &fakeMonitorRepo{m: testMonitor()}, &fakeCheckRepo{}, &fakeIncidentRepo{}
	svc := MonitoringService{Monitors: mr, Checks: cr, Incidents: ir,
		Checker: &scriptChecker{results: []domain.CheckResult{down(500), down(500), down(500)}}}
	var oc *Outcome
	var err error
	for i := 0; i < 3; i++ {
		oc, err = svc.ProcessCheck(context.Background(), "m1")
		if err != nil {
			t.Fatal(err)
		}
	}
	if oc == nil || oc.Transition == nil || oc.Transition.Type != domain.TransitionOpened {
		t.Fatalf("want OPENED transition, got %+v", oc)
	}
	if ir.opened != 1 {
		t.Fatalf("want 1 incident opened, got %d", ir.opened)
	}
	if mr.m.Status != domain.MonitorDown {
		t.Fatalf("want monitor DOWN, got %s", mr.m.Status)
	}
}

func TestProcessCheckResolvesIncident(t *testing.T) {
	mr, cr, ir := &fakeMonitorRepo{m: testMonitor()}, &fakeCheckRepo{}, &fakeIncidentRepo{
		open: &domain.Incident{ID: "inc-1", Status: domain.IncidentOpen},
	}
	svc := MonitoringService{Monitors: mr, Checks: cr, Incidents: ir,
		Checker: &scriptChecker{results: []domain.CheckResult{up(), up()}}}
	var oc *Outcome
	var err error
	for i := 0; i < 2; i++ {
		oc, err = svc.ProcessCheck(context.Background(), "m1")
		if err != nil {
			t.Fatal(err)
		}
	}
	if oc == nil || oc.Transition == nil || oc.Transition.Type != domain.TransitionResolved {
		t.Fatalf("want RESOLVED transition, got %+v", oc)
	}
	if ir.open.RecoveryCount != 2 {
		t.Fatalf("want recovery count 2, got %d", ir.open.RecoveryCount)
	}
}

func TestProcessCheckSkipsInactive(t *testing.T) {
	m := testMonitor()
	m.IsActive = false
	mr := &fakeMonitorRepo{m: m}
	svc := MonitoringService{Monitors: mr, Checks: &fakeCheckRepo{}, Incidents: &fakeIncidentRepo{},
		Checker: &scriptChecker{results: []domain.CheckResult{down(500)}}}
	oc, err := svc.ProcessCheck(context.Background(), "m1")
	if err != nil || oc != nil {
		t.Fatalf("want silent skip, got %+v, %v", oc, err)
	}
}

func TestMonitorCreateValidation(t *testing.T) {
	svc := MonitorService{Monitors: &fakeMonitorRepo{}}
	_, err := svc.Create(context.Background(), "u1", MonitorInput{Name: "x", URL: "http://localhost:9/"})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("want validation error, got %v", err)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	svc := AuthService{Users: stubUserRepo{}, Hash: plainHasher{}, Tokens: fixedToken{}}
	_ = time.Now()
	_, err := svc.Login(context.Background(), "a@b.c", "wrong")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("want unauthorized, got %v", err)
	}
}

type stubUserRepo struct{}

func (stubUserRepo) Create(ctx context.Context, u domain.User) error { return nil }
func (stubUserRepo) ByEmail(ctx context.Context, e string) (domain.User, error) {
	return domain.User{ID: "u1", Email: e, Name: "A", PasswordHash: "secret"}, nil
}
func (stubUserRepo) ByID(ctx context.Context, id string) (domain.User, error) {
	return domain.User{ID: id}, nil
}

type plainHasher struct{}

func (plainHasher) Hash(p string) (string, error) { return p, nil }
func (plainHasher) Compare(h, p string) error {
	if h != p {
		return errors.New("mismatch")
	}
	return nil
}

type fixedToken struct{}

func (fixedToken) Issue(uid string) (string, error) { return "tok-" + uid, nil }
func (fixedToken) Parse(tok string) (string, error) { return "u1", nil }

func strptr(s string) *string { return &s }

func TestMonitorPartialUpdateKeepsThresholds(t *testing.T) {
	repo := &fakeMonitorRepo{m: domain.Monitor{
		ID: "m1", UserID: "u1", Name: "Old", URL: "https://x.com", Method: "GET",
		IntervalSeconds: 120, TimeoutSeconds: 5,
		FailureThreshold: 7, RecoveryThreshold: 9,
		Status: domain.MonitorUp, IsActive: true,
	}}
	svc := MonitorService{Monitors: repo}
	out, err := svc.Update(context.Background(), "m1", "u1", MonitorPatch{Name: strptr("New")})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if out.Name != "New" {
		t.Fatalf("name not updated: %+v", out)
	}
	if out.FailureThreshold != 7 || out.RecoveryThreshold != 9 || out.IntervalSeconds != 120 {
		t.Fatalf("omitted fields reset: %+v", out)
	}
}

func TestMonitorCreateStaggersFirstRun(t *testing.T) {
	repo := &fakeMonitorRepo{}
	svc := MonitorService{Monitors: repo}
	before := time.Now()
	out, err := svc.Create(context.Background(), "u1", MonitorInput{
		Name: "S", URL: "https://x.com", IntervalSeconds: 60,
		TimeoutSeconds: 5, FailureThreshold: 1, RecoveryThreshold: 1,
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	_ = out
	got := repo.m.NextRunAt
	if got == nil {
		t.Fatal("NextRunAt not set on create")
	}
	if got.Before(before) || !got.Before(before.Add(60*time.Second)) {
		t.Fatalf("NextRunAt outside [now, now+interval): %v", got)
	}
}
