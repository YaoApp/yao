// yao-bridge is a lightweight binary that bridges stdio to a Unix socket.
// It is used inside Docker containers to connect CLI tools (like Claude)
// to the Yao IPC server running on the host.
//
// Usage: yao-bridge /tmp/yao.sock
package main

import (
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: yao-bridge <socket-path>")
	}

	sockPath := os.Args[1]

	// Connect to Unix socket
	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		log.Fatalf("Failed to connect to socket %s: %v", sockPath, err)
	}
	defer conn.Close()

	// Handle signals for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Create done channel
	done := make(chan struct{})

	// stdin → socket
	go func() {
		io.Copy(conn, os.Stdin)
		// Close write side when stdin is done
		if unixConn, ok := conn.(*net.UnixConn); ok {
			unixConn.CloseWrite()
		}
	}()

	// socket → stdout
	go func() {
		io.Copy(os.Stdout, conn)
		close(done)
	}()

	// Wait for completion or signal
	select {
	case <-done:
	case <-sigCh:
	}
}
