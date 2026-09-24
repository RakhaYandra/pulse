CREATE TABLE IF NOT EXISTS monitor_checks (
    id BIGSERIAL PRIMARY KEY,
    monitor_id UUID NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL,
    status_code INT NULL,
    response_time_ms INT NULL,
    error_message TEXT NULL,
    checked_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_checks_monitor_time ON monitor_checks(monitor_id, checked_at DESC);
