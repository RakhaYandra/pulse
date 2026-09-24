package read

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/RakhaYandra/pulse/pkg/response"
)

type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *Handler { return &Handler{db: db} }

func uid(c *gin.Context) string { v, _ := c.Get("userID"); return v.(string) }

func (h *Handler) Checks(c *gin.Context) {
	limit := 20
	if q := c.Query("limit"); q != "" {
		if n, err := strconv.Atoi(q); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	var owned int
	err := h.db.QueryRow(`SELECT COUNT(*) FROM monitors WHERE id=$1 AND user_id=$2`,
		c.Param("id"), uid(c)).Scan(&owned)
	if err != nil || owned == 0 {
		response.Err(c, http.StatusNotFound, "monitor not found")
		return
	}
	rows, err := h.db.Query(`SELECT status,status_code,response_time_ms,error_message,checked_at
		FROM monitor_checks WHERE monitor_id=$1 ORDER BY id DESC LIMIT $2`, c.Param("id"), limit)
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var st string
		var code, rt sql.NullInt64
		var em sql.NullString
		var at string
		rows.Scan(&st, &code, &rt, &em, &at)
		out = append(out, gin.H{"status": st, "status_code": nullInt(code),
			"response_time_ms": nullInt(rt), "error": nullStr(em), "checked_at": at})
	}
	response.OK(c, out)
}

func (h *Handler) MonitorIncidents(c *gin.Context) {
	var owned int
	err := h.db.QueryRow(`SELECT COUNT(*) FROM monitors WHERE id=$1 AND user_id=$2`,
		c.Param("id"), uid(c)).Scan(&owned)
	if err != nil || owned == 0 {
		response.Err(c, http.StatusNotFound, "monitor not found")
		return
	}
	h.incidentList(c, `AND i.monitor_id=$2`, c.Param("id"))
}

func (h *Handler) Incidents(c *gin.Context) {
	h.incidentList(c, ``)
}

func (h *Handler) incidentList(c *gin.Context, extra string, args ...any) {
	qargs := append([]any{uid(c)}, args...)
	rows, err := h.db.Query(`SELECT i.id,m.name,i.status,i.reason,i.started_at,i.resolved_at,i.failure_count,i.recovery_count
		FROM incidents i JOIN monitors m ON m.id=i.monitor_id
		WHERE m.user_id=$1 `+extra+` ORDER BY i.started_at DESC LIMIT 100`, qargs...)
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id, name, st, reason, started string
		var resolved sql.NullString
		var fails, recovs int
		rows.Scan(&id, &name, &st, &reason, &started, &resolved, &fails, &recovs)
		out = append(out, gin.H{"id": id, "monitor_name": name, "status": st,
			"reason": reason, "started_at": started, "resolved_at": nullStr(resolved),
			"failure_count": fails, "recovery_count": recovs})
	}
	response.OK(c, out)
}

func (h *Handler) Dashboard(c *gin.Context) {
	u := uid(c)
	var total, up, down, active int
	var overall sql.NullFloat64
	h.db.QueryRow(`SELECT COUNT(*),COUNT(*) FILTER (WHERE status='UP'),COUNT(*) FILTER (WHERE status='DOWN')
		FROM monitors WHERE user_id=$1 AND is_active`, u).Scan(&total, &up, &down)
	h.db.QueryRow(`SELECT COUNT(*) FROM incidents i JOIN monitors m ON m.id=i.monitor_id
		WHERE m.user_id=$1 AND i.status='OPEN'`, u).Scan(&active)
	h.db.QueryRow(`SELECT 100.0*SUM(CASE WHEN status='UP' THEN 1 ELSE 0 END)/NULLIF(COUNT(*),0)
		FROM monitor_checks WHERE monitor_id IN (SELECT id FROM monitors WHERE user_id=$1)
		AND checked_at > now() - interval '24 hours'`, u).Scan(&overall)
	uptime := 0.0
	if overall.Valid {
		uptime = overall.Float64
	}
	response.OK(c, gin.H{"total_monitors": total, "up": up, "down": down,
		"active_incidents": active, "uptime_24h": uptime})
}

func nullInt(n sql.NullInt64) any {
	if n.Valid {
		return n.Int64
	}
	return nil
}

func nullStr(n sql.NullString) any {
	if n.Valid {
		return n.String
	}
	return nil
}
