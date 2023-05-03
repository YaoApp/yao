package neo

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	httpTest "github.com/yaoapp/gou/http"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/helper"
	"github.com/yaoapp/yao/neo/command"
	"github.com/yaoapp/yao/test"
	_ "github.com/yaoapp/yao/utils"
)

func TestAPI(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// test router
	router := testRouter(t)
	err := Neo.API(router, "/neo/chat")
	if err != nil {
		t.Fatal(err)
	}

	// test server
	host, shutdown := testServer(t, router)
	defer shutdown()

	// test request
	url := fmt.Sprintf("%s/neo/chat?content=hello&token=%s", host, testToken(t))
	res := []byte{}
	req := httpTest.New(url).
		WithHeader(http.Header{"Content-Type": []string{"application/json"}})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// send request
	req.Stream(ctx, "GET", nil, func(data []byte) int {
		res = append(res, data...)
		return 1
	})

	assert.Contains(t, string(res), `{"done":true}`)
}

func TestAPIAuth(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	router := testRouter(t)
	err := Neo.API(router, "/neo/chat")
	if err != nil {
		t.Fatal(err)
	}

	response := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/neo/chat?content=hello", nil)
	assert.Panics(t, func() {
		router.ServeHTTP(response, req)
	})
}

func testServer(t *testing.T, router *gin.Engine) (string, func()) {

	// Listen
	l, err := net.Listen("tcp4", ":0")
	if err != nil {
		t.Fatal(err)
	}

	srv := &http.Server{Addr: ":0", Handler: router}

	// start serve
	go func() {
		if err := srv.Serve(l); err != nil && err != http.ErrServerClosed {
			fmt.Println("[TestServer] Error:", err)
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

	// Load Config
	err := Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}

	// Load Commands
	err = command.Load(config.Conf)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	router := gin.New()
	gin.SetMode(gin.ReleaseMode)
	return router
}

func testToken(t *testing.T) string {
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
