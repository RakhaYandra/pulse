package checker

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCheckUp(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()
	res := Check(srv.URL, 5)
	if res.Status != "UP" || res.StatusCode != 200 {
		t.Fatalf("got %+v", res)
	}
	if res.ResponseTimeMs < 0 {
		t.Fatalf("expected non-negative response time, got %+v", res)
	}
}

func TestCheckDown(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()
	res := Check(srv.URL, 5)
	if res.Status != "DOWN" || res.StatusCode != 500 {
		t.Fatalf("got %+v", res)
	}
}

func TestCheckTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(3 * time.Second)
	}))
	defer srv.Close()
	start := time.Now()
	res := Check(srv.URL, 1)
	// 3 attempts x 1s timeout + backoff must still collapse to ONE result
	if res.Status != "TIMEOUT" {
		t.Fatalf("got %+v", res)
	}
	if time.Since(start) > 15*time.Second {
		t.Fatalf("retry took too long")
	}
}

func TestCheckConnectionError(t *testing.T) {
	res := Check("http://127.0.0.1:1/nope", 2)
	if res.Status != "ERROR" {
		t.Fatalf("got %+v", res)
	}
}
