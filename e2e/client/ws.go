package client

import (
	"net/http"

	"github.com/gorilla/websocket"
)

// DialWebSocket opens a WebSocket using the client's base URL and bearer token.
func (c *Client) DialWebSocket(path string) (*websocket.Conn, *http.Response, error) {
	header := http.Header{}
	if c.Token != "" {
		header.Set("Authorization", "Bearer "+c.Token)
	}
	url := "ws" + c.BaseURL[len("http"):] + path
	return websocket.DefaultDialer.Dial(url, header)
}
