package postgres

import (
	"context"
	"database/sql"

	"github.com/RakhaYandra/pulse/internal/domain"
	"github.com/google/uuid"
)

type IncidentRepo struct {
	DB *sql.DB
}

func (r IncidentRepo) Open(ctx context.Context, monitorID, reason string, failCount int) (domain.Incident, error) {
	id := uuid.NewString()
	_, err := r.DB.ExecContext(ctx, `INSERT INTO incidents(id,monitor_id,status,reason,failure_count) VALUES($1,$2,'OPEN',$3,$4)`,
		id, monitorID, reason, failCount)
	if err != nil {
		return domain.Incident{}, err
	}
	return domain.Incident{ID: id, MonitorID: monitorID, Status: domain.IncidentOpen, Reason: reason, FailureCount: failCount}, nil
}

func (r IncidentRepo) OpenFor(ctx context.Context, monitorID string) (domain.Incident, bool, error) {
	var inc domain.Incident
	err := r.DB.QueryRowContext(ctx, `SELECT id,reason,failure_count,started_at FROM incidents
		WHERE monitor_id=$1 AND status='OPEN' ORDER BY started_at DESC LIMIT 1`, monitorID).
		Scan(&inc.ID, &inc.Reason, &inc.FailureCount, &inc.StartedAt)
	if err == sql.ErrNoRows {
		return domain.Incident{}, false, nil
	}
	if err != nil {
		return domain.Incident{}, false, err
	}
	inc.MonitorID = monitorID
	inc.Status = domain.IncidentOpen
	return inc, true, nil
}

func (r IncidentRepo) Resolve(ctx context.Context, id string, recoveryCount int) error {
	_, err := r.DB.ExecContext(ctx, `UPDATE incidents SET status='RESOLVED',resolved_at=now(),recovery_count=$1,updated_at=now() WHERE id=$2`,
		recoveryCount, id)
	return err
}
