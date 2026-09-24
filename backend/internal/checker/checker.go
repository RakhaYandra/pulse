package checker

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

type Result struct {
	Status         string // UP | DOWN | TIMEOUT | ERROR
	StatusCode     int
	ResponseTimeMs int
	ErrorMessage   string
}

// Check executes one monitoring check: up to 3 HTTP attempts (1s,2s backoff)
// but records a single Result. A check is UP only if an attempt returns 2xx.
func Check(targetURL string, timeoutSec int) Result {
	client := &http.Client{Timeout: time.Duration(timeoutSec) * time.Second}
	var lastErr string
	for attempt := 1; attempt <= 3; attempt++ {
		start := time.Now()
		resp, err := client.Get(targetURL)
		elapsed := int(time.Since(start).Milliseconds())
		if err != nil {
			lastErr = err.Error()
			if isTimeout(err) {
				if attempt == 3 {
					return Result{Status: "TIMEOUT", ResponseTimeMs: elapsed, ErrorMessage: lastErr}
				}
			} else if attempt == 3 {
				return Result{Status: "ERROR", ResponseTimeMs: elapsed, ErrorMessage: lastErr}
			}
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return Result{Status: "UP", StatusCode: resp.StatusCode, ResponseTimeMs: elapsed}
		}
		return Result{Status: "DOWN", StatusCode: resp.StatusCode, ResponseTimeMs: elapsed,
			ErrorMessage: fmt.Sprintf("unexpected status %d", resp.StatusCode)}
	}
	return Result{Status: "ERROR", ErrorMessage: lastErr}
}

func isTimeout(err error) bool {
	if e, ok := err.(interface{ Timeout() bool }); ok && e.Timeout() {
		return true
	}
	return false
}
