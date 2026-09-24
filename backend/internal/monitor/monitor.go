package monitor

import (
	"database/sql"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/RakhaYandra/pulse/pkg/response"
)

type Monitor struct {
	ID                string  `json:"id"`
	Name              string  `json:"name"`
	URL               string  `json:"url"`
	Method            string  `json:"method"`
	IntervalSeconds   int     `json:"interval_seconds"`
	TimeoutSeconds    int     `json:"timeout_seconds"`
	FailureThreshold  int     `json:"failure_threshold"`
	RecoveryThreshold int     `json:"recovery_threshold"`
	Status            string  `json:"status"`
	IsActive          bool    `json:"is_active"`
	LastCheckedAt     *string `json:"last_checked_at"`
}

type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *Handler { return &Handler{db: db} }

var cols = `id,name,url,method,interval_seconds,timeout_seconds,failure_threshold,recovery_threshold,status,is_active,last_checked_at`

func scanRow(s interface {
	Scan(...any) error
}) (Monitor, error) {
	var m Monitor
	var last sql.NullString
	err := s.Scan(&m.ID, &m.Name, &m.URL, &m.Method, &m.IntervalSeconds,
		&m.TimeoutSeconds, &m.FailureThreshold, &m.RecoveryThreshold,
		&m.Status, &m.IsActive, &last)
	if last.Valid {
		m.LastCheckedAt = &last.String
	}
	return m, err
}

type input struct {
	Name              string `json:"name"`
	URL               string `json:"url"`
	IntervalSeconds   int    `json:"interval_seconds"`
	TimeoutSeconds    int    `json:"timeout_seconds"`
	FailureThreshold  int    `json:"failure_threshold"`
	RecoveryThreshold int    `json:"recovery_threshold"`
}

func (in *input) normalize() {
	if in.IntervalSeconds == 0 {
		in.IntervalSeconds = 300
	}
	if in.TimeoutSeconds == 0 {
		in.TimeoutSeconds = 5
	}
	if in.FailureThreshold == 0 {
		in.FailureThreshold = 3
	}
	if in.RecoveryThreshold == 0 {
		in.RecoveryThreshold = 2
	}
	in.Name = strings.TrimSpace(in.Name)
	in.URL = strings.TrimSpace(in.URL)
}

func (in *input) validate() string {
	if in.Name == "" {
		return "name required"
	}
	u, err := url.ParseRequestURI(in.URL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "url must be valid http(s)"
	}
	host := strings.ToLower(u.Hostname())
	if host == "localhost" || strings.HasPrefix(host, "127.") || host == "::1" {
		return "url host not allowed"
	}
	if in.IntervalSeconds < 60 {
		return "interval_seconds min 60"
	}
	if in.TimeoutSeconds < 1 || in.TimeoutSeconds > 60 {
		return "timeout_seconds 1..60"
	}
	if in.TimeoutSeconds >= in.IntervalSeconds {
		return "timeout must be < interval"
	}
	if in.FailureThreshold < 1 || in.RecoveryThreshold < 1 {
		return "thresholds min 1"
	}
	return ""
}

func uid(c *gin.Context) string { v, _ := c.Get("userID"); return v.(string) }

func (h *Handler) Create(c *gin.Context) {
	var in input
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Err(c, http.StatusBadRequest, "invalid body")
		return
	}
	in.normalize()
	if msg := in.validate(); msg != "" {
		response.Err(c, http.StatusBadRequest, msg)
		return
	}
	id := uuid.NewString()
	_, err := h.db.Exec(`INSERT INTO monitors(id,user_id,name,url,method,interval_seconds,timeout_seconds,failure_threshold,recovery_threshold)
		VALUES($1,$2,$3,$4,'GET',$5,$6,$7,$8)`,
		id, uid(c), in.Name, in.URL, in.IntervalSeconds, in.TimeoutSeconds, in.FailureThreshold, in.RecoveryThreshold)
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "create failed")
		return
	}
	m, err := h.get(id, uid(c))
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "create failed")
		return
	}
	response.OK(c, m)
}

func (h *Handler) get(id, userID string) (Monitor, error) {
	return scanRow(h.db.QueryRow(`SELECT `+cols+` FROM monitors WHERE id=$1 AND user_id=$2`, id, userID))
}

func (h *Handler) List(c *gin.Context) {
	rows, err := h.db.Query(`SELECT `+cols+` FROM monitors WHERE user_id=$1 ORDER BY created_at DESC`, uid(c))
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "list failed")
		return
	}
	defer rows.Close()
	out := []Monitor{}
	for rows.Next() {
		m, err := scanRow(rows)
		if err != nil {
			response.Err(c, http.StatusInternalServerError, "list failed")
			return
		}
		out = append(out, m)
	}
	response.OK(c, out)
}

func (h *Handler) Get(c *gin.Context) {
	m, err := h.get(c.Param("id"), uid(c))
	if err == sql.ErrNoRows {
		response.Err(c, http.StatusNotFound, "monitor not found")
		return
	}
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "get failed")
		return
	}
	response.OK(c, m)
}

func (h *Handler) Update(c *gin.Context) {
	var in input
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Err(c, http.StatusBadRequest, "invalid body")
		return
	}
	in.normalize()
	if msg := in.validate(); msg != "" {
		response.Err(c, http.StatusBadRequest, msg)
		return
	}
	res, err := h.db.Exec(`UPDATE monitors SET name=$1,url=$2,interval_seconds=$3,timeout_seconds=$4,
		failure_threshold=$5,recovery_threshold=$6,updated_at=now() WHERE id=$7 AND user_id=$8`,
		in.Name, in.URL, in.IntervalSeconds, in.TimeoutSeconds, in.FailureThreshold, in.RecoveryThreshold,
		c.Param("id"), uid(c))
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "update failed")
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		response.Err(c, http.StatusNotFound, "monitor not found")
		return
	}
	m, _ := h.get(c.Param("id"), uid(c))
	response.OK(c, m)
}

func (h *Handler) Delete(c *gin.Context) {
	res, err := h.db.Exec(`DELETE FROM monitors WHERE id=$1 AND user_id=$2`, c.Param("id"), uid(c))
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "delete failed")
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		response.Err(c, http.StatusNotFound, "monitor not found")
		return
	}
	response.OK(c, gin.H{"deleted": c.Param("id")})
}

func (h *Handler) setActive(active bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		res, err := h.db.Exec(`UPDATE monitors SET is_active=$1,updated_at=now() WHERE id=$2 AND user_id=$3`,
			active, c.Param("id"), uid(c))
		if err != nil {
			response.Err(c, http.StatusInternalServerError, "update failed")
			return
		}
		if n, _ := res.RowsAffected(); n == 0 {
			response.Err(c, http.StatusNotFound, "monitor not found")
			return
		}
		m, _ := h.get(c.Param("id"), uid(c))
		response.OK(c, m)
	}
}

func (h *Handler) Pause(c *gin.Context)  { h.setActive(false)(c) }
func (h *Handler) Resume(c *gin.Context) { h.setActive(true)(c) }
