# ADR-003 — Retry attempt ≠ monitoring check

Status: accepted. Context: naive retry-as-new-check inflates check counts and
corrupts uptime math (3 attempts = 3 failures).
Decision: up to 3 HTTP attempts (backoff 1s, 2s) collapse into ONE
`monitor_checks` row and ONE incident-evaluation step.
Uptime = UP rows / total rows, unaffected by attempt count.
