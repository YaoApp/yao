package neo

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	httpTest "github.com/yaoapp/gou/http"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/helper"
	"github.com/yaoapp/yao/test"
)

func init() {
	// Set gin to release mode to reduce log output
	gin.SetMode(gin.ReleaseMode)
}

func TestAPI(t *testing.T) {
	// Disable test logging
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Redirect stdout to /dev/null
	oldStdout := os.Stdout
	null, _ := os.Open(os.DevNull)
	os.Stdout = null
	defer func() {
		os.Stdout = oldStdout
		null.Close()
	}()

	// test router
	router := testRouter(t)
	err := Neo.API(router, "/neo/chat")
	if err != nil {
		t.Fatal(err)
	}

	// test server
	host, shutdown := testServer(t, router)
	defer shutdown()

	tests := []struct {
		name       string
		url        string
		method     string
		headers    http.Header
		expectCode int
		expectBody string
	}{
		{
			name:       "Basic Chat Request",
			url:        fmt.Sprintf("/neo/chat?content=hello&token=%s", testToken()),
			method:     "GET",
			headers:    http.Header{"Content-Type": []string{"application/json"}},
			expectBody: `{`,
		},
		{
			name:       "Chat with System Message",
			url:        fmt.Sprintf("/neo/chat?content=hello&system=You are a helpful assistant&token=%s", testToken()),
			method:     "GET",
			headers:    http.Header{"Content-Type": []string{"application/json"}},
			expectBody: `{`,
		},
		{
			name:       "Chat with Model Parameter",
			url:        fmt.Sprintf("/neo/chat?content=hello&model=gpt-3.5-turbo&token=%s", testToken()),
			method:     "GET",
			headers:    http.Header{"Content-Type": []string{"application/json"}},
			expectBody: `{`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := fmt.Sprintf("%s%s", host, tt.url)
			res := []byte{}
			req := httpTest.New(url).WithHeader(tt.headers)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			req.Stream(ctx, tt.method, nil, func(data []byte) int {
				res = append(res, data...)
				return 1
			})

			assert.Contains(t, string(res), tt.expectBody)
		})
	}
}

func TestAPIAuth(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Redirect stdout and stderr to /dev/null
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	null, _ := os.Open(os.DevNull)
	os.Stdout = null
	os.Stderr = null
	defer func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
		null.Close()
	}()

	router := testRouter(t)
	err := Neo.API(router, "/neo/chat")
	if err != nil {
		t.Fatal(err)
	}

	// Separate tests for authentication errors and parameter validation errors
	authTests := []struct {
		name       string
		url        string
		method     string
		expectCode int
	}{
		{
			name:       "Missing Token",
			url:        "/neo/chat?content=hello",
			method:     "GET",
			expectCode: http.StatusUnauthorized,
		},
		{
			name:       "Invalid Token",
			url:        "/neo/chat?content=hello&token=invalid",
			method:     "GET",
			expectCode: http.StatusUnauthorized,
		},
	}

	// Test authentication errors (will panic)
	for _, tt := range authTests {
		t.Run(tt.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			req, _ := http.NewRequest(tt.method, tt.url, nil)
			assert.Panics(t, func() {
				router.ServeHTTP(response, req)
			})
		})
	}

	// Test parameter validation errors (will return status code)
	validationTests := []struct {
		name       string
		url        string
		method     string
		expectCode int
	}{
		{
			name:       "Missing Content",
			url:        fmt.Sprintf("/neo/chat?token=%s", testToken()),
			method:     "GET",
			expectCode: http.StatusBadRequest,
		},
	}

	// Test parameter validation errors (return status code)
	for _, tt := range validationTests {
		t.Run(tt.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			req, _ := http.NewRequest(tt.method, tt.url, nil)
			router.ServeHTTP(response, req)
			assert.Equal(t, tt.expectCode, response.Code)
		})
	}
}

// Helper functions
func testServer(t *testing.T, router *gin.Engine) (string, func()) {
	l, err := net.Listen("tcp4", ":0")
	if err != nil {
		t.Fatal(err)
	}

	srv := &http.Server{Addr: ":0", Handler: router}

	go func() {
		if err := srv.Serve(l); err != nil && err != http.ErrServerClosed {
			return
		}
	}()

	addr := strings.Split(l.Addr().String(), ":")
	if len(addr) != 2 {
		t.Fatal("invalid address")
	}

	host := fmt.Sprintf("http://127.0.0.1:%s", addr[1])
	time.Sleep(50 * time.Millisecond)

	shutdown := func() {
		srv.Close()
		l.Close()
	}
	return host, shutdown
}

func testRouter(t *testing.T) *gin.Engine {
	err := Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}

	router := gin.New() // Use gin.New() instead of gin.Default() to avoid default logging middleware
	return router
}

func testToken() string {
	token := helper.JwtMake(1,
		map[string]interface{}{
			"id":   1,
			"name": "Test",
		},
		map[string]interface{}{
			"exp": 3600,
			"sid": "123456",
		})
	return token.Token
}
