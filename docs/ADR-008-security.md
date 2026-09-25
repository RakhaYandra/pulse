# ADR-008 — Security & prod hardening (SSRF, rate limit, secrets, PATCH)

Status: accepted (2026-09-25).

## SSRF
`domain.BlockedHost/BlockedIP` rejects loopback, private (10/8, 172.16/12,
192.168/16, fc00::/7), link-local incl. 169.254.169.254, unspecified,
multicast. `infrastructure/httpcheck` resolves once per attempt and dials
the resolved IP (TOCTOU-safe, SNI preserved); redirects never followed
(3xx recorded DOWN). `PULSE_ALLOW_HOSTS` exempts names (bench `stub`);
default empty = strict. Bench seed bypasses the API, so workers need the
allowlist in compose.

## Rate limiting
`delivery/http` token bucket per IP (`x/time/rate`): auth 10/min (burst 10),
global API 100/min (burst 100), 429 + Retry-After. In-memory → per-replica
budgets when scaled; Redis bucket is future work.

## Secrets & CORS
`JWT_SECRET` has no default — api mode FATALs without it; compose uses
`${JWT_SECRET:?...}` so even `up` fails loudly. `CORS_ALLOWED_ORIGINS`
defaults to `http://localhost:5173`; `*` only when set explicitly. Grafana
password via env (dev default documented).

## True PATCH
`MonitorPatch` pointer fields: omitted keys keep values (previously reset to
defaults via `normalize()`). FE sends full objects — unaffected.
`ResponseTimeMs` redefined as wall-clock incl. backoff (amends ADR-003).
Shared `http.Transport` (connection reuse) with per-request timeout ctx.
