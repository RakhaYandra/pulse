ALTER TABLE monitors ADD COLUMN IF NOT EXISTS next_run_at TIMESTAMPTZ NULL;
-- Due is now computed from scheduled time, not completion time.
DROP INDEX IF EXISTS idx_monitors_due;
CREATE INDEX IF NOT EXISTS idx_monitors_next_run ON monitors(is_active, next_run_at);
