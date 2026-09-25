package http

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/RakhaYandra/pulse/domain"
	"github.com/RakhaYandra/pulse/usecase"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	Auth       usecase.AuthService
	Monitors   usecase.MonitorService
	Monitoring usecase.MonitoringService // reserved for future check-trigger endpoints
	Dash       usecase.DashboardService
	Tokens     usecase.TokenIssuer
}

func uid(c *gin.Context) string {
	v, _ := c.Get("userID")
	s, _ := v.(string)
	return s
}

// --- auth ---

func (h Handler) Register(c *gin.Context) {
	var in registerRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	out, err := h.Auth.Register(c.Request.Context(), in.Email, in.Password, in.Name)
	if err != nil {
		writeErr(c, err)
		return
	}
	writeOK(c, gin.H{"token": out.Token, "user": userResponse(out.User)})
}

func (h Handler) Login(c *gin.Context) {
	var in loginRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	out, err := h.Auth.Login(c.Request.Context(), in.Email, in.Password)
	if err != nil {
		writeErr(c, err)
		return
	}
	writeOK(c, gin.H{"token": out.Token, "user": userResponse(out.User)})
}

func (h Handler) Me(c *gin.Context) {
	u, err := h.Auth.Me(c.Request.Context(), uid(c))
	if err != nil {
		writeNotFound(c, "user")
		return
	}
	writeOK(c, userResponse(u))
}

// --- monitors ---

func toInput(in monitorRequest) usecase.MonitorInput {
	return usecase.MonitorInput{
		Name: in.Name, URL: in.URL,
		IntervalSeconds: in.IntervalSeconds, TimeoutSeconds: in.TimeoutSeconds,
		FailureThreshold: in.FailureThreshold, RecoveryThreshold: in.RecoveryThreshold,
	}
}

func toMonitorResponse(m usecase.MonitorDTO) monitorResponse {
	return monitorResponse{
		ID: m.ID, Name: m.Name, URL: m.URL, Method: m.Method,
		IntervalSeconds: m.IntervalSeconds, TimeoutSeconds: m.TimeoutSeconds,
		FailureThreshold: m.FailureThreshold, RecoveryThreshold: m.RecoveryThreshold,
		Status: string(m.Status), IsActive: m.IsActive,
		LastCheckedAt: fmtTimePtr(m.LastCheckedAt),
	}
}

func (h Handler) CreateMonitor(c *gin.Context) {
	var in monitorRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	m, err := h.Monitors.Create(c.Request.Context(), uid(c), toInput(in))
	if err != nil {
		writeErr(c, err)
		return
	}
	writeOK(c, toMonitorResponse(m))
}

func (h Handler) ListMonitors(c *gin.Context) {
	ms, err := h.Monitors.List(c.Request.Context(), uid(c))
	if err != nil {
		writeErr(c, err)
		return
	}
	out := make([]monitorResponse, 0, len(ms))
	for _, m := range ms {
		out = append(out, toMonitorResponse(m))
	}
	writeOK(c, out)
}

func (h Handler) GetMonitor(c *gin.Context) {
	m, err := h.Monitors.Get(c.Request.Context(), c.Param("id"), uid(c))
	if err != nil {
		writeNotFound(c, "monitor")
		return
	}
	writeOK(c, toMonitorResponse(m))
}

func (h Handler) UpdateMonitor(c *gin.Context) {
	var in monitorPatchRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	m, err := h.Monitors.Update(c.Request.Context(), c.Param("id"), uid(c), usecase.MonitorPatch{
		Name: in.Name, URL: in.URL,
		IntervalSeconds: in.IntervalSeconds, TimeoutSeconds: in.TimeoutSeconds,
		FailureThreshold: in.FailureThreshold, RecoveryThreshold: in.RecoveryThreshold,
	})
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeNotFound(c, "monitor")
			return
		}
		writeErr(c, err)
		return
	}
	writeOK(c, toMonitorResponse(m))
}

func (h Handler) DeleteMonitor(c *gin.Context) {
	if err := h.Monitors.Delete(c.Request.Context(), c.Param("id"), uid(c)); err != nil {
		writeNotFound(c, "monitor")
		return
	}
	writeOK(c, gin.H{"deleted": c.Param("id")})
}

func (h Handler) setActive(c *gin.Context, active bool) {
	m, err := h.Monitors.SetActive(c.Request.Context(), c.Param("id"), uid(c), active)
	if err != nil {
		writeNotFound(c, "monitor")
		return
	}
	writeOK(c, toMonitorResponse(m))
}

func (h Handler) PauseMonitor(c *gin.Context)  { h.setActive(c, false) }
func (h Handler) ResumeMonitor(c *gin.Context) { h.setActive(c, true) }

// --- reads ---

func (h Handler) Checks(c *gin.Context) {
	limit := 20
	if q := c.Query("limit"); q != "" {
		if n, err := strconv.Atoi(q); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	cs, err := h.Dash.Checks(c.Request.Context(), c.Param("id"), uid(c), limit)
	if err != nil {
		writeNotFound(c, "monitor")
		return
	}
	out := make([]checkResponse, 0, len(cs))
	for _, v := range cs {
		out = append(out, checkResponse{
			Status: string(v.Status), StatusCode: v.StatusCode,
			ResponseTimeMs: v.ResponseTimeMs, Error: v.Error,
			CheckedAt: fmtTime(v.CheckedAt),
		})
	}
	writeOK(c, out)
}

func toIncidentResponse(v usecase.IncidentView) incidentResponse {
	return incidentResponse{
		ID: v.ID, MonitorID: v.MonitorID, MonitorName: v.MonitorName, Status: string(v.Status),
		Reason: v.Reason, StartedAt: fmtTime(v.StartedAt),
		ResolvedAt:   fmtTimePtr(v.ResolvedAt),
		FailureCount: v.FailureCount, RecoveryCount: v.RecoveryCount,
	}
}

func (h Handler) MonitorIncidents(c *gin.Context) {
	ins, err := h.Dash.Incidents(c.Request.Context(), uid(c), c.Param("id"))
	if err != nil {
		writeNotFound(c, "monitor")
		return
	}
	out := make([]incidentResponse, 0, len(ins))
	for _, v := range ins {
		out = append(out, toIncidentResponse(v))
	}
	writeOK(c, out)
}

func (h Handler) Incidents(c *gin.Context) {
	ins, err := h.Dash.Incidents(c.Request.Context(), uid(c), "")
	if err != nil {
		writeErr(c, err)
		return
	}
	out := make([]incidentResponse, 0, len(ins))
	for _, v := range ins {
		out = append(out, toIncidentResponse(v))
	}
	writeOK(c, out)
}

func (h Handler) Dashboard(c *gin.Context) {
	s, err := h.Dash.Summary(c.Request.Context(), uid(c))
	if err != nil {
		writeErr(c, err)
		return
	}
	writeOK(c, summaryResponse{
		TotalMonitors: s.TotalMonitors, Up: s.Up, Down: s.Down,
		ActiveIncidents: s.ActiveIncidents, Uptime24h: s.Uptime24h,
	})
}

// --- middleware ---

func (h Handler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			c.Abort()
			return
		}
		sub, err := h.Tokens.Parse(strings.TrimPrefix(header, "Bearer "))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}
		c.Set("userID", sub)
		c.Next()
	}
}
