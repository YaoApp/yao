package log

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	accessWriter      *lumberjack.Logger
	accessErrorWriter *lumberjack.Logger
)

// InitAccessLog initializes access log and access-error log writers.
// Must be called before any HTTP request is served.
func InitAccessLog(root string) {
	logDir := filepath.Join(root, "logs")
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		os.MkdirAll(logDir, 0755)
	}

	accessWriter = &lumberjack.Logger{
		Filename:   filepath.Join(logDir, "access.log"),
		MaxSize:    100,
		MaxBackups: 5,
		MaxAge:     30,
		LocalTime:  true,
	}
	accessErrorWriter = &lumberjack.Logger{
		Filename:   filepath.Join(logDir, "access-error.log"),
		MaxSize:    50,
		MaxBackups: 5,
		MaxAge:     30,
		LocalTime:  true,
	}
}

// AccessLog returns a gin middleware that writes NGINX Combined Log Format
// to access.log (all requests) and access-error.log (4xx/5xx only).
func AccessLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if accessWriter == nil {
			return
		}

		status := c.Writer.Status()
		size := c.Writer.Size()
		if size < 0 {
			size = 0
		}

		line := fmt.Sprintf("%s - %s [%s] \"%s %s %s\" %d %d \"%s\" \"%s\"\n",
			c.ClientIP(),
			remoteUser(c),
			time.Now().Format("02/Jan/2006:15:04:05 -0700"),
			c.Request.Method,
			c.Request.RequestURI,
			c.Request.Proto,
			status,
			size,
			dash(c.Request.Referer()),
			dash(c.Request.UserAgent()),
		)

		accessWriter.Write([]byte(line))
		if status >= 400 {
			accessErrorWriter.Write([]byte(line))
		}
	}
}

// remoteUser extracts a user identifier from the gin context.
// Tries __username (JWT), __user_id (OAuth), __sid (SUI session) in order.
func remoteUser(c *gin.Context) string {
	for _, key := range []string{"__username", "__user_id", "__sid"} {
		if v, ok := c.Get(key); ok {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
	}
	return "-"
}

func dash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
