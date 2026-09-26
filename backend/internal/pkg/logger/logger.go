package logger

import (
	"log"
	"os"
	"strings"
)

type Logger struct {
	l *log.Logger
}

func New(level string) *Logger {
	return &Logger{l: log.New(os.Stdout, "", log.LstdFlags)}
}

func (l *Logger) Info(msg string, kv ...any) {
	l.l.Println("[INFO]", msg, strings.TrimSpace(kvString(kv)))
}

func (l *Logger) Error(msg string, kv ...any) {
	l.l.Println("[ERROR]", msg, strings.TrimSpace(kvString(kv)))
}

func kvString(kv []any) string {
	if len(kv) == 0 {
		return ""
	}
	s := ""
	for i := 0; i+1 < len(kv); i += 2 {
		s += " " + anyString(kv[i]) + "=" + anyString(kv[i+1])
	}
	return s
}

func anyString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case error:
		return t.Error()
	default:
		return ""
	}
}
