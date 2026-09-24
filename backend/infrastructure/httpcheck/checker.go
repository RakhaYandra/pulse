package httpcheck

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/RakhaYandra/pulse/domain"
)

type Checker struct{}

// Check executes one monitoring check: up to 3 HTTP attempts (1s,2s backoff)
// but records a single CheckResult. UP only on 2xx (ADR-003).
func (Checker) Check(targetURL string, timeoutSec int) domain.CheckResult {
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

func isTimeout(err error) bool {
	if e, ok := err.(interface{ Timeout() bool }); ok && e.Timeout() {
		return true
	}
	return false
}
