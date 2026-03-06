package proxy

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

// --- Remote Connect ---

func (r *remoteProxy) Connect(ctx context.Context, containerID string, opts ConnectOptions) (*Connection, error) {
	baseURL, err := r.URL(ctx, containerID, opts.Port, opts.Path)
	if err != nil {
		return nil, err
	}
	return connect(ctx, baseURL, opts.Protocol, r.client)
}

// --- Tunnel Connect ---

func (t *tunnelProxy) Connect(ctx context.Context, containerID string, opts ConnectOptions) (*Connection, error) {
	baseURL, err := t.URL(ctx, containerID, opts.Port, opts.Path)
	if err != nil {
		return nil, err
	}
	return connect(ctx, baseURL, opts.Protocol, http.DefaultClient)
}

// --- Local Connect ---

func (l *localProxy) Connect(ctx context.Context, containerID string, opts ConnectOptions) (*Connection, error) {
	baseURL, err := l.URL(ctx, containerID, opts.Port, opts.Path)
	if err != nil {
		return nil, err
	}
	return connect(ctx, baseURL, opts.Protocol, http.DefaultClient)
}

func connect(ctx context.Context, url string, protocol string, hc *http.Client) (*Connection, error) {
	switch protocol {
	case "ws":
		return connectWS(ctx, url)
	case "sse":
		return connectSSE(ctx, url, hc)
	default:
		return nil, fmt.Errorf("unsupported connect protocol: %q", protocol)
	}
}

func connectWS(ctx context.Context, rawURL string) (*Connection, error) {
	wsURL := strings.Replace(rawURL, "http://", "ws://", 1)
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("ws dial: %w", err)
	}

	ch := make(chan []byte, 64)
	go func() {
		defer close(ch)
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			ch <- msg
		}
	}()

	return &Connection{
		Messages: ch,
		Send: func(data []byte) error {
			return conn.WriteMessage(websocket.TextMessage, data)
		},
		Close: func() error {
			return conn.Close()
		},
	}, nil
}

func connectSSE(ctx context.Context, url string, hc *http.Client) (*Connection, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/event-stream")

	resp, err := hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sse connect: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("sse: status %d", resp.StatusCode)
	}

	ch := make(chan []byte, 64)
	go func() {
		defer close(ch)
		defer resp.Body.Close()
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")
				ch <- bytes.Clone([]byte(data))
			}
		}
	}()

	return &Connection{
		Messages: ch,
		Send: func(data []byte) error {
			return fmt.Errorf("sse: send not supported")
		},
		Close: func() error {
			resp.Body.Close()
			return nil
		},
	}, nil
}
