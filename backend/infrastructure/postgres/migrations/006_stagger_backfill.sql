-- One-time spread of existing schedules: same deterministic stagger as new
-- creates (offset from id hash), so pre-existing synchronized monitors stop
-- firing as one wave. MarkScheduled overwrites per enqueue afterwards.
UPDATE monitors
SET next_run_at = now() + (abs(hashtext(id::text) % interval_seconds) || ' seconds')::interval;
