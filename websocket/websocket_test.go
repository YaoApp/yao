package websocket

import (
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/websocket"
	"github.com/yaoapp/yao/config"
)

func TestLoad(t *testing.T) {
	Load(config.Conf)
	check(t)
}

func TestWebSocketOpen(t *testing.T) {
	// Load(config.Conf)
	// script.Load(config.Conf)
	// srv, url := serve(t)
	// defer srv.Stop()

	// ws := websocket.Se("message")
	// err := ws.Open(url, "messageV2", "chatV3")
	// if err != nil {
	// 	t.Fatal(err)
	// }
}

func serve(t *testing.T) (*websocket.Upgrader, string) {

	ws, err := websocket.NewUpgrader("test")
	if err != nil {
		t.Fatalf("%s", err)
	}

	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	ws.SetHandler(func(message []byte, id int) ([]byte, error) { return message, nil })
	ws.SetRouter(router)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	go ws.Start()
	go func() {
		http.Serve(listener, router)
	}()
	time.Sleep(200 * time.Millisecond)

	return ws, fmt.Sprintf("ws://127.0.0.1:%d/websocket/test", listener.Addr().(*net.TCPAddr).Port)
}

func check(t *testing.T) {
	// keys := []string{}
	// for key := range gou.WebSockets {
	// 	keys = append(keys, key)
	// }
	// assert.Equal(t, 1, len(keys))
}
