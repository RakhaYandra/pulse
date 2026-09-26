package domain

import (
	"net"
	"testing"
	"time"
)

func TestMonitorValidate(t *testing.T) {
	good := Monitor{Name: "P", URL: "https://api.example.com/h", IntervalSeconds: 300,
		TimeoutSeconds: 5, FailureThreshold: 3, RecoveryThreshold: 2}
	if err := good.Validate(); err != nil {
		t.Fatalf("good monitor rejected: %v", err)
	}
	cases := []Monitor{
		{Name: "", URL: "https://x.com"},
		{Name: "P", URL: "not-a-url"},
		{Name: "P", URL: "http://localhost:9/x", IntervalSeconds: 300, TimeoutSeconds: 5, FailureThreshold: 1, RecoveryThreshold: 1},
		{Name: "P", URL: "http://10.0.0.1/x", IntervalSeconds: 300, TimeoutSeconds: 5, FailureThreshold: 1, RecoveryThreshold: 1},
		{Name: "P", URL: "http://192.168.1.1/x", IntervalSeconds: 300, TimeoutSeconds: 5, FailureThreshold: 1, RecoveryThreshold: 1},
		{Name: "P", URL: "http://172.16.0.5/x", IntervalSeconds: 300, TimeoutSeconds: 5, FailureThreshold: 1, RecoveryThreshold: 1},
		{Name: "P", URL: "http://169.254.169.254/x", IntervalSeconds: 300, TimeoutSeconds: 5, FailureThreshold: 1, RecoveryThreshold: 1},
		{Name: "P", URL: "http://[::1]/x", IntervalSeconds: 300, TimeoutSeconds: 5, FailureThreshold: 1, RecoveryThreshold: 1},
		{Name: "P", URL: "http://[fd00::1]/x", IntervalSeconds: 300, TimeoutSeconds: 5, FailureThreshold: 1, RecoveryThreshold: 1},
		{Name: "P", URL: "https://x.com", IntervalSeconds: 10, TimeoutSeconds: 5, FailureThreshold: 1, RecoveryThreshold: 1},
		{Name: "P", URL: "https://x.com", IntervalSeconds: 60, TimeoutSeconds: 60, FailureThreshold: 1, RecoveryThreshold: 1},
	}
	for i, m := range cases {
		if m.IntervalSeconds == 0 {
			m.IntervalSeconds = 300
		}
		if m.TimeoutSeconds == 0 {
			m.TimeoutSeconds = 5
		}
		if m.FailureThreshold == 0 {
			m.FailureThreshold = 3
		}
		if m.RecoveryThreshold == 0 {
			m.RecoveryThreshold = 2
		}
		if err := m.Validate(); err == nil {
			t.Fatalf("case %d accepted, want validation error", i)
		}
	}
}

func TestBlockedIP(t *testing.T) {
	blocked := []string{"127.0.0.1", "10.1.2.3", "172.16.5.4", "192.168.0.1", "169.254.169.254", "::1", "fd00::1", "0.0.0.0"}
	for _, s := range blocked {
		if !BlockedIP(net.ParseIP(s)) {
			t.Fatalf("%s should be blocked", s)
		}
	}
	allowed := []string{"8.8.8.8", "1.1.1.1", "93.184.216.34"}
	for _, s := range allowed {
		if BlockedIP(net.ParseIP(s)) {
			t.Fatalf("%s should be allowed", s)
		}
	}
}

func TestEvalStreak(t *testing.T) {
	f, o := EvalStreak([]CheckStatus{CheckDown, CheckDown, CheckDown})
	if f != 3 || o != 0 {
		t.Fatalf("got fail=%d ok=%d", f, o)
	}
	f, o = EvalStreak([]CheckStatus{CheckUp, CheckUp, CheckDown})
	if f != 0 || o != 2 {
		t.Fatalf("got fail=%d ok=%d", f, o)
	}
	f, o = EvalStreak(nil)
	if f != 0 || o != 0 {
		t.Fatalf("got fail=%d ok=%d", f, o)
	}
}

func TestStaggerInitialRunAt(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	interval := 60 * time.Second
	a := StaggerInitialRunAt("id-1", interval, now)
	if a.Before(now) || !a.Before(now.Add(interval)) {
		t.Fatalf("offset out of [0, interval): %v", a.Sub(now))
	}
	b := StaggerInitialRunAt("id-1", interval, now)
	if !a.Equal(b) {
		t.Fatal("same ID must yield same offset")
	}
	seen := map[time.Duration]bool{}
	for i := 0; i < 50; i++ {
		seen[StaggerInitialRunAt("id-"+string(rune('a'+i)), interval, now).Sub(now)] = true
	}
	if len(seen) < 10 {
		t.Fatalf("offsets do not spread: only %d distinct", len(seen))
	}
	if got := StaggerInitialRunAt("x", 0, now); !got.Equal(now) {
		t.Fatal("non-positive interval must return now")
	}
}
