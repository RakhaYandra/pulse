package httpcheck

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/RakhaYandra/pulse/domain"
)

func TestCheckUp(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()
	res := (Checker{}).Check(srv.URL, 5)
	if res.Status != domain.CheckUp || res.StatusCode != 200 {
		t.Fatalf("got %+v", res)
	}
	if res.ResponseTimeMs < 0 {
		t.Fatalf("negative response time: %+v", res)
	}
}

func TestCheckDown(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()
	res := (Checker{}).Check(srv.URL, 5)
	if res.Status != domain.CheckDown || res.StatusCode != 500 {
		t.Fatalf("got %+v", res)
	}
}

func TestCheckTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(3 * time.Second)
	}))
	defer srv.Close()
	start := time.Now()
	res := (Checker{}).Check(srv.URL, 1)
	if res.Status != domain.CheckTimeout {
		t.Fatalf("got %+v", res)
	}
	if time.Since(start) > 15*time.Second {
		t.Fatalf("retry took too long")
	}
}

func TestCheckConnectionError(t *testing.T) {
	res := (Checker{}).Check("http://127.0.0.1:1/nope", 2)
	if res.Status != domain.CheckError {
		t.Fatalf("got %+v", res)
	}
}
