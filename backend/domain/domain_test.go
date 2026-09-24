package domain

import "testing"

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
