package log

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupTestLog(t *testing.T) (string, func()) {
	t.Helper()
	dir := t.TempDir()
	InitAccessLog(dir)
	return dir, func() {
		if accessWriter != nil {
			accessWriter.Close()
		}
		if accessErrorWriter != nil {
			accessErrorWriter.Close()
		}
		accessWriter = nil
		accessErrorWriter = nil
	}
}

// NGINX Combined: $remote_addr - $remote_user [$time_local] "$request" $status $body_bytes_sent "$http_referer" "$http_user_agent"
var nginxCombinedRe = regexp.MustCompile(
	`^(\S+) - (\S+) \[\d{2}/\w{3}/\d{4}:\d{2}:\d{2}:\d{2} [+-]\d{4}\] "(\S+) (\S+) (\S+)" (\d{3}) (\d+) "(.*)" "(.*)"$`,
)

func TestAccessLog_NginxFormat(t *testing.T) {
	dir, cleanup := setupTestLog(t)
	defer cleanup()

	router := gin.New()
	router.Use(AccessLog())
	router.GET("/api/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("User-Agent", "TestAgent/1.0")
	req.Header.Set("Referer", "https://example.com")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	data, err := os.ReadFile(filepath.Join(dir, "logs", "access.log"))
	if err != nil {
		t.Fatalf("read access.log: %v", err)
	}

	line := strings.TrimSpace(string(data))
	if !nginxCombinedRe.MatchString(line) {
		t.Errorf("access.log line does not match NGINX Combined format:\n%s", line)
	}

	if !strings.Contains(line, `"GET /api/test HTTP/1.1"`) {
		t.Errorf("expected request line in log, got: %s", line)
	}
	if !strings.Contains(line, `"https://example.com"`) {
		t.Errorf("expected referer in log, got: %s", line)
	}
	if !strings.Contains(line, `"TestAgent/1.0"`) {
		t.Errorf("expected user-agent in log, got: %s", line)
	}

	// access-error.log should be empty for 200
	errData, err := os.ReadFile(filepath.Join(dir, "logs", "access-error.log"))
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("read access-error.log: %v", err)
	}
	if len(strings.TrimSpace(string(errData))) > 0 {
		t.Errorf("access-error.log should be empty for 200, got: %s", string(errData))
	}
}

func TestAccessLog_ErrorDoubleWrite(t *testing.T) {
	dir, cleanup := setupTestLog(t)
	defer cleanup()

	router := gin.New()
	router.Use(AccessLog())
	router.GET("/api/fail", func(c *gin.Context) {
		c.String(http.StatusInternalServerError, "error")
	})
	router.GET("/api/notfound", func(c *gin.Context) {
		c.String(http.StatusNotFound, "not found")
	})

	// 500 request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/api/fail", nil))

	// 404 request
	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/api/notfound", nil))

	// 200 request (should NOT appear in error log)
	router.GET("/api/ok", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/api/ok", nil))

	accessData, _ := os.ReadFile(filepath.Join(dir, "logs", "access.log"))
	accessLines := nonEmptyLines(string(accessData))
	if len(accessLines) != 3 {
		t.Fatalf("access.log: expected 3 lines, got %d:\n%s", len(accessLines), string(accessData))
	}

	errData, _ := os.ReadFile(filepath.Join(dir, "logs", "access-error.log"))
	errLines := nonEmptyLines(string(errData))
	if len(errLines) != 2 {
		t.Fatalf("access-error.log: expected 2 lines (500+404), got %d:\n%s", len(errLines), string(errData))
	}

	if !strings.Contains(errLines[0], "500") {
		t.Errorf("first error line should contain 500: %s", errLines[0])
	}
	if !strings.Contains(errLines[1], "404") {
		t.Errorf("second error line should contain 404: %s", errLines[1])
	}
}

func TestAccessLog_RemoteUser(t *testing.T) {
	dir, cleanup := setupTestLog(t)
	defer cleanup()

	router := gin.New()
	router.Use(AccessLog())
	router.GET("/api/user", func(c *gin.Context) {
		c.Set("__username", "alice")
		c.String(http.StatusOK, "ok")
	})
	router.GET("/api/userid", func(c *gin.Context) {
		c.Set("__user_id", "uid-123")
		c.String(http.StatusOK, "ok")
	})
	router.GET("/api/anon", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	for _, path := range []string{"/api/user", "/api/userid", "/api/anon"} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
	}

	data, _ := os.ReadFile(filepath.Join(dir, "logs", "access.log"))
	lines := nonEmptyLines(string(data))
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}

	// Note: AccessLog middleware runs c.Next() first, then reads context.
	// The user keys are set inside the handler which runs during c.Next(),
	// so they should be available when the log line is written.
	if !strings.Contains(lines[0], " alice ") {
		t.Errorf("line 1 should have user 'alice': %s", lines[0])
	}
	if !strings.Contains(lines[1], " uid-123 ") {
		t.Errorf("line 2 should have user 'uid-123': %s", lines[1])
	}
	if !strings.Contains(lines[2], " - ") {
		t.Errorf("line 3 should have '-' for anonymous: %s", lines[2])
	}
}

func TestAccessLog_DashForEmpty(t *testing.T) {
	dir, cleanup := setupTestLog(t)
	defer cleanup()

	router := gin.New()
	router.Use(AccessLog())
	router.GET("/api/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest("GET", "/api/test", nil)
	// No Referer, no User-Agent
	req.Header.Del("User-Agent")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	data, _ := os.ReadFile(filepath.Join(dir, "logs", "access.log"))
	line := strings.TrimSpace(string(data))

	// Should end with "-" "-" for empty referer and user-agent
	if !strings.HasSuffix(line, `"-" "-"`) {
		t.Errorf("expected dash for empty referer/ua, got: %s", line)
	}
}

func nonEmptyLines(s string) []string {
	var result []string
	for _, line := range strings.Split(s, "\n") {
		if strings.TrimSpace(line) != "" {
			result = append(result, line)
		}
	}
	return result
}
