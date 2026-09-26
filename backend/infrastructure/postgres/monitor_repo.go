package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/RakhaYandra/pulse/domain"
)

type MonitorRepo struct {
	DB *sql.DB
}

const monitorCols = `id,user_id,name,url,method,interval_seconds,timeout_seconds,failure_threshold,recovery_threshold,status,is_active,last_checked_at,next_run_at,created_at`

func scanMonitor(s interface{ Scan(...any) error }) (domain.Monitor, error) {
	var m domain.Monitor
	var status string
	var last, next sql.NullTime
	var created sql.NullTime
	err := s.Scan(&m.ID, &m.UserID, &m.Name, &m.URL, &m.Method, &m.IntervalSeconds,
		&m.TimeoutSeconds, &m.FailureThreshold, &m.RecoveryThreshold,
		&status, &m.IsActive, &last, &next, &created)
	if err != nil {
		return domain.Monitor{}, err
	}
	m.Status = domain.MonitorStatus(status)
	if last.Valid {
		m.LastCheckedAt = &last.Time
	}
	if next.Valid {
		m.NextRunAt = &next.Time
	}
	if created.Valid {
		m.CreatedAt = created.Time
	}
	return m, nil
}

func (r MonitorRepo) Create(ctx context.Context, m domain.Monitor) error {
	_, err := r.DB.ExecContext(ctx, `INSERT INTO monitors(id,user_id,name,url,method,interval_seconds,timeout_seconds,failure_threshold,recovery_threshold,next_run_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,COALESCE($10,now()))`,
		m.ID, m.UserID, m.Name, m.URL, m.Method, m.IntervalSeconds, m.TimeoutSeconds, m.FailureThreshold, m.RecoveryThreshold, nullTime(m.NextRunAt))
	return err
}

func (r MonitorRepo) ByID(ctx context.Context, id, userID string) (domain.Monitor, error) {
	m, err := scanMonitor(r.DB.QueryRowContext(ctx, `SELECT `+monitorCols+` FROM monitors WHERE id=$1 AND user_id=$2`, id, userID))
	if err == sql.ErrNoRows {
		return domain.Monitor{}, domain.ErrNotFound
	}
	return m, err
}

func (r MonitorRepo) ListByUser(ctx context.Context, userID string) ([]domain.Monitor, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT `+monitorCols+` FROM monitors WHERE user_id=$1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Monitor{}
	for rows.Next() {
		m, err := scanMonitor(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func nullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

func (r MonitorRepo) Update(ctx context.Context, m domain.Monitor) error {
	res, err := r.DB.ExecContext(ctx, `UPDATE monitors SET name=$1,url=$2,interval_seconds=$3,timeout_seconds=$4,
		failure_threshold=$5,recovery_threshold=$6,next_run_at=COALESCE($9,next_run_at),updated_at=now() WHERE id=$7 AND user_id=$8`,
		m.Name, m.URL, m.IntervalSeconds, m.TimeoutSeconds, m.FailureThreshold, m.RecoveryThreshold, m.ID, m.UserID, nullTime(m.NextRunAt))
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r MonitorRepo) Delete(ctx context.Context, id, userID string) error {
	res, err := r.DB.ExecContext(ctx, `DELETE FROM monitors WHERE id=$1 AND user_id=$2`, id, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r MonitorRepo) SetActive(ctx context.Context, id, userID string, active bool) (domain.Monitor, error) {
	res, err := r.DB.ExecContext(ctx, `UPDATE monitors SET is_active=$1,updated_at=now() WHERE id=$2 AND user_id=$3`,
		active, id, userID)
	if err != nil {
		return domain.Monitor{}, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.Monitor{}, domain.ErrNotFound
	}
	return r.ByID(ctx, id, userID)
}

func (r MonitorRepo) FindDue(ctx context.Context) ([]domain.Monitor, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT `+monitorCols+` FROM monitors WHERE is_active
		AND (next_run_at IS NULL OR next_run_at < now())`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Monitor{}
	for rows.Next() {
		m, err := scanMonitor(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r MonitorRepo) ActiveMonitor(ctx context.Context, id string) (domain.Monitor, error) {
	m, err := scanMonitor(r.DB.QueryRowContext(ctx, `SELECT `+monitorCols+` FROM monitors WHERE id=$1 AND is_active`, id))
	if err == sql.ErrNoRows {
		return domain.Monitor{}, domain.ErrNotFound
	}
	return m, err
}

func (r MonitorRepo) RecordStatus(ctx context.Context, id string, st domain.MonitorStatus) error {
	_, err := r.DB.ExecContext(ctx, `UPDATE monitors SET status=$1,last_checked_at=now(),updated_at=now() WHERE id=$2`,
		string(st), id)
	return err
}

// MarkScheduled advances the next due time from the scheduled moment, so
// cadence doesn't drift with drain time.
func (r MonitorRepo) MarkScheduled(ctx context.Context, id string, next time.Time) error {
	_, err := r.DB.ExecContext(ctx, `UPDATE monitors SET next_run_at=$1 WHERE id=$2`, next, id)
	return err
}
