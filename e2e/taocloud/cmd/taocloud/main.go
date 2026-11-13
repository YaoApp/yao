// Command taocloud runs the Tao Computers contract mock for UAT/integration.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yaoapp/yao/e2e/taocloud"
)

func main() {
	addr := os.Getenv("TAO_MOCK_ADDR")
	if addr == "" {
		addr = "127.0.0.1:18080"
	}
	server := taocloud.New()
	server.HTTP.Close() // standalone mode uses its own http.Server

	httpServer := &http.Server{Addr: addr, Handler: server.Handler()}
	fmt.Printf("tao mock listening on http://%s\n", addr)
	fmt.Printf("test api key: %s\n", server.Token)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdown)
	}()
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
