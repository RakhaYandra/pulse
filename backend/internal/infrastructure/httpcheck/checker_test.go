package httpcheck

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/RakhaYandra/pulse/internal/domain"
)

func TestCheckUp(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()
	res := (Checker{Allow: map[string]bool{"127.0.0.1": true}}).Check(srv.URL, 5)
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
	res := (Checker{Allow: map[string]bool{"127.0.0.1": true}}).Check(srv.URL, 5)
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
	res := (Checker{Allow: map[string]bool{"127.0.0.1": true}}).Check(srv.URL, 1)
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

func TestCheckBlockedLiteral(t *testing.T) {
	for _, target := range []string{"http://10.0.0.1/", "http://192.168.1.1/", "http://169.254.169.254/"} {
		res := (Checker{}).Check(target, 2)
		if res.Status != domain.CheckError {
			t.Fatalf("%s: got %+v", target, res)
		}
	}
}

func TestPickIP(t *testing.T) {
	priv := func(host string) ([]net.IP, error) { return []net.IP{net.ParseIP("10.9.9.9")}, nil }
	if _, err := pickIP("internal.local", priv, nil); err == nil {
		t.Fatal("private resolve should be blocked")
	}
	ip, err := pickIP("internal.local", priv, map[string]bool{"internal.local": true})
	if err != nil || ip.String() != "10.9.9.9" {
		t.Fatalf("allowlisted host should pass: %v %v", ip, err)
	}
	mixed := func(host string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("10.0.0.1"), net.ParseIP("93.184.216.34")}, nil
	}
	ip, err = pickIP("dual.example", mixed, nil)
	if err != nil || ip.String() != "93.184.216.34" {
		t.Fatalf("should pick public IP: %v %v", ip, err)
	}
}

func TestRedirectNotFollowed(t *testing.T) {
	final := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer final.Close()
	redir := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, final.URL, http.StatusFound)
	}))
	defer redir.Close()
	allow := map[string]bool{"127.0.0.1": true}
	res := (Checker{Allow: allow}).Check(redir.URL, 5)
	if res.Status != domain.CheckDown || res.StatusCode != 302 {
		t.Fatalf("redirect should record 302 DOWN, got %+v", res)
	}
}
