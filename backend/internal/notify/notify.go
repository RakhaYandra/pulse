package notify

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/RakhaYandra/pulse/pkg/logger"
)

type Sender struct {
	log     *logger.Logger
	token   string
	chatID  string
	client  *http.Client
	enabled bool
}

func New(log *logger.Logger) *Sender {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	chat := os.Getenv("TELEGRAM_CHAT_ID")
	return &Sender{
		log:     log,
		token:   token,
		chatID:  chat,
		client:  &http.Client{Timeout: 10 * time.Second},
		enabled: token != "" && chat != "",
	}
}

// Incident sends OPEN/RESOLVED notifications. No-op when not configured.
func (s *Sender) Incident(text string) {
	if !s.enabled {
		s.log.Info("notify skipped (telegram not configured)", "text", text)
		return
	}
	api := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", s.token)
	resp, err := s.client.PostForm(api, url.Values{"chat_id": {s.chatID}, "text": {text}})
	if err != nil {
		s.log.Error("telegram send failed", "err", err)
		return
	}
	resp.Body.Close()
	if resp.StatusCode >= 300 {
		s.log.Error("telegram send bad status", "status", resp.Status)
	}
}
