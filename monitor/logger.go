package monitor

import (
	"log/slog"
	"os"
	"path/filepath"

	"gopkg.in/natefinch/lumberjack.v2"
)

const slogLevelTrace = slog.Level(-8)

var logger *slog.Logger

// initLogger creates the monitor logger.
// appMode controls the minimum log level:
//   - "production"  → Info (Trace alerts are not written)
//   - "development" → Trace (everything is written)
func initLogger(root string, logMode string, appMode string) {
	logDir := filepath.Join(root, "logs")
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		os.MkdirAll(logDir, 0755)
	}

	w := &lumberjack.Logger{
		Filename:   filepath.Join(logDir, "monitor.log"),
		MaxSize:    50,
		MaxBackups: 3,
		MaxAge:     7,
		LocalTime:  true,
	}

	minLevel := slog.LevelInfo
	if appMode == "development" {
		minLevel = slogLevelTrace
	}

	opts := &slog.HandlerOptions{Level: minLevel}
	var handler slog.Handler
	if logMode == "JSON" {
		handler = slog.NewJSONHandler(w, opts)
	} else {
		handler = slog.NewTextHandler(w, opts)
	}

	logger = slog.New(handler)
}

func levelToSlog(l Level) slog.Level {
	switch l {
	case Trace:
		return slogLevelTrace
	case Warn:
		return slog.LevelWarn
	case Error:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
