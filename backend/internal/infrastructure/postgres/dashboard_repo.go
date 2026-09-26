package postgres

import (
	"context"
	"database/sql"

	"github.com/RakhaYandra/pulse/internal/domain"
	"github.com/RakhaYandra/pulse/internal/usecase"
)

type DashboardRepo struct {
	DB *sql.DB
}

func (r DashboardRepo) Summary(ctx context.Context, userID string) (usecase.Summary, error) {
	var s usecase.Summary
	var overall sql.NullFloat64
	if err := r.DB.QueryRowContext(ctx, `SELECT COUNT(*),COUNT(*) FILTER (WHERE status='UP'),COUNT(*) FILTER (WHERE status='DOWN')
		FROM monitors WHERE user_id=$1 AND is_active`, userID).Scan(&s.TotalMonitors, &s.Up, &s.Down); err != nil {
		return s, err
	}
	if err := r.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM incidents i JOIN monitors m ON m.id=i.monitor_id
		WHERE m.user_id=$1 AND i.status='OPEN'`, userID).Scan(&s.ActiveIncidents); err != nil {
		return s, err
	}
	if err := r.DB.QueryRowContext(ctx, `SELECT 100.0*SUM(CASE WHEN status='UP' THEN 1 ELSE 0 END)/NULLIF(COUNT(*),0)
		FROM monitor_checks WHERE monitor_id IN (SELECT id FROM monitors WHERE user_id=$1)
		AND checked_at > now() - interval '24 hours'`, userID).Scan(&overall); err != nil {
		return s, err
	}
	if overall.Valid {
		s.Uptime24h = overall.Float64
	}
	return s, nil
}

func (r DashboardRepo) Checks(ctx context.Context, monitorID, userID string, limit int) ([]usecase.CheckView, error) {
	var owned int
	err := r.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM monitors WHERE id=$1 AND user_id=$2`,
		monitorID, userID).Scan(&owned)
	if err != nil {
		return nil, err
	}
	if owned == 0 {
		return nil, domain.ErrNotFound
	}
	rows, err := r.DB.QueryContext(ctx, `SELECT status,status_code,response_time_ms,error_message,checked_at
		FROM monitor_checks WHERE monitor_id=$1 ORDER BY id DESC LIMIT $2`, monitorID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []usecase.CheckView{}
	for rows.Next() {
		var v usecase.CheckView
		var st string
		var code, rt sql.NullInt64
		var em sql.NullString
		if err := rows.Scan(&st, &code, &rt, &em, &v.CheckedAt); err != nil {
			return nil, err
		}
		v.Status = domain.CheckStatus(st)
		if code.Valid {
			c := int(code.Int64)
			v.StatusCode = &c
		}
		if rt.Valid {
			t := int(rt.Int64)
			v.ResponseTimeMs = &t
		}
		if em.Valid {
			v.Error = em.String
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r DashboardRepo) Incidents(ctx context.Context, userID, monitorID string, limit int) ([]usecase.IncidentView, error) {
	q := `SELECT i.id,i.monitor_id,m.name,i.status,i.reason,i.started_at,i.resolved_at,i.failure_count,i.recovery_count
		FROM incidents i JOIN monitors m ON m.id=i.monitor_id
		WHERE m.user_id=$1 `
	args := []any{userID}
	if monitorID != "" {
		var owned int
		err := r.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM monitors WHERE id=$1 AND user_id=$2`,
			monitorID, userID).Scan(&owned)
		if err != nil {
			return nil, err
		}
		if owned == 0 {
			return nil, domain.ErrNotFound
		}
		q += `AND i.monitor_id=$2 `
		args = append(args, monitorID)
	}
	q += `ORDER BY i.started_at DESC LIMIT 100`
	rows, err := r.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []usecase.IncidentView{}
	for rows.Next() {
		var v usecase.IncidentView
		var st string
		var resolved sql.NullTime
		if err := rows.Scan(&v.ID, &v.MonitorID, &v.MonitorName, &st, &v.Reason, &v.StartedAt, &resolved, &v.FailureCount, &v.RecoveryCount); err != nil {
			return nil, err
		}
		v.Status = domain.IncidentStatus(st)
		if resolved.Valid {
			v.ResolvedAt = &resolved.Time
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r DashboardRepo) Reliability(ctx context.Context, userID string, days int) ([]usecase.ReliabilityRow, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT m.id, m.name,
		COUNT(i.id) FILTER (WHERE i.started_at > now() - make_interval(days => $2)),
		COUNT(i.id) FILTER (WHERE i.status='OPEN'),
		AVG(EXTRACT(EPOCH FROM (i.resolved_at - i.started_at))) FILTER (WHERE i.resolved_at IS NOT NULL AND i.started_at > now() - make_interval(days => $2)),
		COALESCE((SELECT 100.0*SUM(CASE WHEN c.status='UP' THEN 1 ELSE 0 END)/NULLIF(COUNT(*),0)
			FROM monitor_checks c WHERE c.monitor_id=m.id AND c.checked_at > now() - make_interval(days => $2)), 0),
		COALESCE((SELECT COUNT(*) FROM monitor_checks c WHERE c.monitor_id=m.id AND c.checked_at > now() - make_interval(days => $2)), 0)
		FROM monitors m LEFT JOIN incidents i ON i.monitor_id=m.id
		WHERE m.user_id=$1 GROUP BY m.id, m.name ORDER BY m.name`, userID, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []usecase.ReliabilityRow{}
	for rows.Next() {
		var v usecase.ReliabilityRow
		var mttr sql.NullFloat64
		var up float64
		if err := rows.Scan(&v.MonitorID, &v.MonitorName, &v.IncidentsTotal, &v.IncidentsOpen, &mttr, &up, &v.ChecksTotal); err != nil {
			return nil, err
		}
		if mttr.Valid {
			v.MTTRSeconds = &mttr.Float64
		}
		v.UptimePct = up
		v.WindowDays = days
		out = append(out, v)
	}
	return out, rows.Err()
}
