package httpcheck

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/RakhaYandra/pulse/domain"
)

// Checker executes HTTP monitoring checks with SSRF protection:
//   - the target hostname is resolved once per attempt; resolved IPs are
//     rejected when private/loopback/link-local unless the host is allowlisted
//     (PULSE_ALLOW_HOSTS, e.g. bench stub on Docker-internal DNS)
//   - redirects are NOT followed (each hop could point internal); 3xx is
//     recorded as DOWN like any other unexpected status
//   - the dialed IP is the resolved one (TOCTOU-safe); TLS ServerName stays
//     the original hostname
type Checker struct {
	Allow map[string]bool
	// Resolve is swappable in tests. Defaults to the system resolver.
	Resolve func(host string) ([]net.IP, error)
}

// Check executes one monitoring check: up to 3 HTTP attempts (1s,2s backoff)
// but records a single CheckResult. UP only on 2xx (ADR-003).
func (c Checker) Check(targetURL string, timeoutSec int) domain.CheckResult {
	u, err := url.ParseRequestURI(targetURL)
	if err != nil {
		return domain.CheckResult{Status: domain.CheckError, ErrorMessage: "invalid url"}
	}
	// Fast path: literal blocked hosts rejected without any DNS or attempts.
	if domain.BlockedHost(u.Hostname()) && !c.Allow[strings.ToLower(u.Hostname())] {
		return domain.CheckResult{Status: domain.CheckError, ErrorMessage: "url host not allowed"}
	}
	resolve := c.Resolve
	if resolve == nil {
		resolve = func(host string) ([]net.IP, error) {
			return net.DefaultResolver.LookupIP(context.Background(), "ip", host)
		}
	}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			ip, err := pickIP(host, resolve, c.Allow)
			if err != nil {
				return nil, err
			}
			return (&net.Dialer{Timeout: time.Duration(timeoutSec) * time.Second}).DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		},
		TLSHandshakeTimeout: time.Duration(timeoutSec) * time.Second,
	}
	client := &http.Client{
		Timeout:       time.Duration(timeoutSec) * time.Second,
		Transport:     transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse },
	}
	var lastErr string
	for attempt := 1; attempt <= 3; attempt++ {
		start := time.Now()
		resp, err := client.Get(targetURL)
		elapsed := int(time.Since(start).Milliseconds())
		if err != nil {
			lastErr = err.Error()
			if isTimeout(err) {
				if attempt == 3 {
					return domain.CheckResult{Status: domain.CheckTimeout, ResponseTimeMs: elapsed, ErrorMessage: lastErr}
				}
			} else if attempt == 3 {
				return domain.CheckResult{Status: domain.CheckError, ResponseTimeMs: elapsed, ErrorMessage: lastErr}
			}
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return domain.CheckResult{Status: domain.CheckUp, StatusCode: resp.StatusCode, ResponseTimeMs: elapsed}
		}
		return domain.CheckResult{Status: domain.CheckDown, StatusCode: resp.StatusCode, ResponseTimeMs: elapsed,
			ErrorMessage: fmt.Sprintf("unexpected status %d", resp.StatusCode)}
	}
	return domain.CheckResult{Status: domain.CheckError, ErrorMessage: lastErr}
}

// pickIP resolves host and returns the first non-blocked IP, unless the host
// is explicitly allowlisted. Pure except for the injected resolver.
func pickIP(host string, resolve func(string) ([]net.IP, error), allow map[string]bool) (net.IP, error) {
	ips, err := resolve(host)
	if err != nil || len(ips) == 0 {
		return nil, fmt.Errorf("cannot resolve %s", host)
	}
	if allow[strings.ToLower(host)] {
		return ips[0], nil
	}
	for _, ip := range ips {
		if !domain.BlockedIP(ip) {
			return ip, nil
		}
	}
	return nil, fmt.Errorf("host resolves to blocked address: %s", host)
}

func isTimeout(err error) bool {
	if e, ok := err.(interface{ Timeout() bool }); ok && e.Timeout() {
		return true
	}
	return false
}
