package notify

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/RakhaYandra/pulse/domain"
	"github.com/RakhaYandra/pulse/pkg/logger"
)

type TelegramNotifier struct {
	Log     *logger.Logger
	Token   string
	ChatID  string
	Enabled bool
}

func (s TelegramNotifier) Notify(t domain.Transition) {
	var text string
	switch t.Type {
	case domain.TransitionOpened:
		text = fmt.Sprintf("🔴 INCIDENT OPENED\n%s\n%s\n%d consecutive failures (last: %s)",
			t.MonitorName, t.MonitorURL, t.FailureCount, t.Detail)
	case domain.TransitionResolved:
		text = fmt.Sprintf("🟢 INCIDENT RESOLVED\n%s\n%s\n%d consecutive successes",
			t.MonitorName, t.MonitorURL, t.SuccessCount)
	default:
		return
	}
	if !s.Enabled {
		s.Log.Info("notify skipped (telegram not configured)", "text", text)
		return
	}
	api := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", s.Token)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.PostForm(api, url.Values{"chat_id": {s.ChatID}, "text": {text}})
	if err != nil {
		s.Log.Error("telegram send failed", "err", err)
		return
	}
	resp.Body.Close()
	if resp.StatusCode >= 300 {
		s.Log.Error("telegram send bad status", "status", resp.Status)
	}
}
