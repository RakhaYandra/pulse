package postgres

import (
	"context"
	"database/sql"

	"github.com/RakhaYandra/pulse/internal/domain"
)

type CheckRepo struct {
	DB *sql.DB
}

func nullInt64(v int) sql.NullInt64 {
	if v == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(v), Valid: true}
}

func (r CheckRepo) Append(ctx context.Context, monitorID string, res domain.CheckResult) error {
	var msg sql.NullString
	if res.ErrorMessage != "" {
		msg = sql.NullString{String: res.ErrorMessage, Valid: true}
	}
	_, err := r.DB.ExecContext(ctx, `INSERT INTO monitor_checks(monitor_id,status,status_code,response_time_ms,error_message)
		VALUES($1,$2,$3,$4,$5)`, monitorID, string(res.Status), nullInt64(res.StatusCode), nullInt64(res.ResponseTimeMs), msg)
	return err
}

func (r CheckRepo) RecentStatuses(ctx context.Context, monitorID string, limit int) ([]domain.CheckStatus, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT status FROM monitor_checks WHERE monitor_id=$1 ORDER BY id DESC LIMIT $2`,
		monitorID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.CheckStatus{}
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, domain.CheckStatus(s))
	}
	return out, rows.Err()
}
