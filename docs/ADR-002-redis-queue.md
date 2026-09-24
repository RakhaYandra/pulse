# ADR-002 — Redis as job queue

Status: accepted. Context: monitoring must not depend on dashboard request
lifecycle; checks must survive API restarts and parallelize across workers.
Decision: scheduler LPUSHes job JSON to `pulse:jobs`; workers BLPop it.
Redis holds no business state — jobs are re-derivable from `monitors`
(due-scan), so queue loss only delays checks, never corrupts data.
