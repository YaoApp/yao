// Package client contains E2E HTTP clients.
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is a small HTTP client scoped to one E2E environment.
type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

// New creates a client for baseURL.
func New(baseURL string) *Client {
	return &Client{BaseURL: baseURL, HTTP: &http.Client{Timeout: 30 * time.Second}}
}

// Do sends a request.
func (c *Client) Do(req *http.Request) (*http.Response, []byte, error) {
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	return resp, body, err
}

// Get performs GET.
func (c *Client) Get(path string) (*http.Response, []byte, error) {
	req, err := http.NewRequest(http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return nil, nil, err
	}
	return c.Do(req)
}

// PostJSON performs POST with a JSON body.
func (c *Client) PostJSON(path string, payload interface{}) (*http.Response, []byte, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal request: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, c.BaseURL+path, bytes.NewReader(data))
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.Do(req)
}
