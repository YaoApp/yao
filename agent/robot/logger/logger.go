package logger

import (
	"fmt"

	kunlog "github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
)

const (
	Reset     = "\033[0m"
	Red       = "\033[31m"
	Green     = "\033[32m"
	Yellow    = "\033[33m"
	Cyan      = "\033[36m"
	White     = "\033[37m"
	Gray      = "\033[90m"
	BoldCyan  = "\033[1;36m"
	BoldGreen = "\033[1;32m"
	BoldRed   = "\033[1;31m"

	reset  = Reset
	red    = Red
	yellow = Yellow
	cyan   = Cyan
	gray   = Gray
)

// Logger provides robot-level structured logging. All integration adapters,
// dispatchers, event handlers, etc. share this implementation.
//
// Dev mode  → colored stdout + kun/log.Trace (unified).
// Prod mode → kun/log at matching level.
type Logger struct {
	tag string
}

// New creates a Logger tagged with the given component name
// (e.g. "telegram", "dispatcher", "message", "delivery").
func New(tag string) *Logger {
	return &Logger{tag: tag}
}

func (l *Logger) prefix() string {
	return fmt.Sprintf("[robot:%s]", l.tag)
}

func (l *Logger) Trace(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if config.IsDevelopment() && !config.Silent {
		fmt.Printf("%s  → %s %s%s\n", gray, l.prefix(), msg, reset)
	}
	kunlog.Trace("%s %s", l.prefix(), msg)
}

func (l *Logger) Debug(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if config.IsDevelopment() && !config.Silent {
		fmt.Printf("%s  • %s %s%s\n", gray, l.prefix(), msg, reset)
	}
	kunlog.Debug("%s %s", l.prefix(), msg)
}

func (l *Logger) Info(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if config.IsDevelopment() && !config.Silent {
		fmt.Printf("%s  ℹ %s %s%s\n", cyan, l.prefix(), msg, reset)
	}
	kunlog.Info("%s %s", l.prefix(), msg)
}

func (l *Logger) Warn(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if config.IsDevelopment() && !config.Silent {
		fmt.Printf("%s  ⚠ %s %s%s\n", yellow, l.prefix(), msg, reset)
	}
	kunlog.Warn("%s %s", l.prefix(), msg)
}

func (l *Logger) Error(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if config.IsDevelopment() && !config.Silent {
		fmt.Printf("%s  ✗ %s %s%s\n", red, l.prefix(), msg, reset)
	}
	kunlog.Error("%s %s", l.prefix(), msg)
}

// IsDev returns true when running in development mode.
func IsDev() bool {
	return config.IsDevelopment()
}

// Raw writes pre-formatted text directly to stdout in dev mode only.
// Use for rich multi-line output (box-style logs, tables, etc.)
// that should bypass the standard single-line prefix format.
func Raw(s string) {
	if config.IsDevelopment() && !config.Silent {
		fmt.Print(s)
	}
}
