package incident

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/RakhaYandra/pulse/internal/checker"
)

// Evaluate stores one check row, updates monitor status, and opens/resolves
// incidents based on failure/recovery thresholds.
// Returns "OPENED", "RESOLVED", or "" plus a notify text when a transition happens.
func Evaluate(db *sql.DB, monitorID string, res checker.Result) (string, string, error) {
	var failureTh, recoveryTh int
	var name, url string
	err := db.QueryRow(`SELECT name,url,failure_threshold,recovery_threshold FROM monitors WHERE id=$1`,
		monitorID).Scan(&name, &url, &failureTh, &recoveryTh)
	if err != nil {
		return "", "", err
	}

	code := sql.NullInt64{}
	if res.StatusCode != 0 {
		code = sql.NullInt64{Int64: int64(res.StatusCode), Valid: true}
	}
	rt := sql.NullInt64{}
	if res.ResponseTimeMs != 0 {
		rt = sql.NullInt64{Int64: int64(res.ResponseTimeMs), Valid: true}
	}
	errMsg := sql.NullString{}
	if res.ErrorMessage != "" {
		errMsg = sql.NullString{String: res.ErrorMessage, Valid: true}
	}
	if _, err := db.Exec(`INSERT INTO monitor_checks(monitor_id,status,status_code,response_time_ms,error_message)
		VALUES($1,$2,$3,$4,$5)`, monitorID, res.Status, code, rt, errMsg); err != nil {
		return "", "", err
	}

	success := res.Status == "UP"
	newStatus := "UP"
	if !success {
		newStatus = "DOWN"
	}
	if _, err := db.Exec(`UPDATE monitors SET status=$1,last_checked_at=now(),updated_at=now() WHERE id=$2`,
		newStatus, monitorID); err != nil {
		return "", "", err
	}

	// Count consecutive trailing failures/successes (bounded scan, cheap for MVP).
	rows, err := db.Query(`SELECT status FROM monitor_checks WHERE monitor_id=$1 ORDER BY id DESC LIMIT $2`,
		monitorID, max(failureTh, recoveryTh)+1)
	if err != nil {
		return "", "", err
	}
	defer rows.Close()
	failStreak, okStreak := 0, 0
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return "", "", err
		}
		if s == "UP" {
			if failStreak > 0 {
				break
			}
			okStreak++
		} else {
			if okStreak > 0 {
				break
			}
			failStreak++
		}
	}
	rows.Close()

	var openID string
	err = db.QueryRow(`SELECT id FROM incidents WHERE monitor_id=$1 AND status='OPEN' ORDER BY started_at DESC LIMIT 1`,
		monitorID).Scan(&openID)
	hasOpen := err == nil

	if !success && failStreak >= failureTh && !hasOpen {
		id := uuid.NewString()
		reason := fmt.Sprintf("%d consecutive failures (last: %s)", failStreak, describe(res))
		if _, err := db.Exec(`INSERT INTO incidents(id,monitor_id,status,reason,failure_count) VALUES($1,$2,'OPEN',$3,$4)`,
			id, monitorID, reason, failStreak); err != nil {
			return "", "", err
		}
		return "OPENED", fmt.Sprintf("🔴 INCIDENT OPENED\n%s\n%s\n%s", name, url, reason), nil
	}
	if success && okStreak >= recoveryTh && hasOpen {
		if _, err := db.Exec(`UPDATE incidents SET status='RESOLVED',resolved_at=now(),recovery_count=$1,updated_at=now() WHERE id=$2`,
			okStreak, openID); err != nil {
			return "", "", err
		}
		return "RESOLVED", fmt.Sprintf("🟢 INCIDENT RESOLVED\n%s\n%s\n%d consecutive successes", name, url, okStreak), nil
	}
	return "", "", nil
}

func describe(res checker.Result) string {
	if res.StatusCode != 0 {
		return fmt.Sprintf("status %d in %dms", res.StatusCode, res.ResponseTimeMs)
	}
	return fmt.Sprintf("%s: %s", res.Status, res.ErrorMessage)
}
