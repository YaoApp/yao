package computer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Client is the Tao Computers HTTP client.
type Client struct {
	Endpoint string
	HTTP     *http.Client
}

// NewClient creates a Tao Computers client from endpoint/env.
func NewClient(endpoint string) *Client {
	if endpoint == "" {
		endpoint = os.Getenv("TAO_ENDPOINT")
	}
	timeout := 30 * time.Second
	if raw := os.Getenv("TAO_TIMEOUT"); raw != "" {
		if d, err := time.ParseDuration(raw); err == nil {
			timeout = d
		}
	}
	return &Client{Endpoint: strings.TrimRight(endpoint, "/"), HTTP: &http.Client{Timeout: timeout}}
}

// Create creates a Tao cloud host.
func (c *Client) Create(ctx context.Context, taoKey string, req CreateInstanceRequest) (*Instance, error) {
	var out Instance
	if err := c.doWithIdempotency(ctx, http.MethodPost, PathComputers, taoKey, req, &out, req.IdempotencyKey); err != nil {
		return nil, err
	}
	return &out, nil
}

// Get returns one Tao cloud host.
func (c *Client) Get(ctx context.Context, taoKey, instanceID string) (*Instance, error) {
	var out Instance
	if err := c.do(ctx, http.MethodGet, PathComputers+"/"+url.PathEscape(instanceID), taoKey, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// List returns Tao cloud hosts.
func (c *Client) List(ctx context.Context, taoKey string, opts ListOptions) ([]Instance, error) {
	q := url.Values{}
	if opts.Status != "" {
		q.Set("status", opts.Status)
	}
	if opts.Region != "" {
		q.Set("region", opts.Region)
	}
	if opts.Page > 0 {
		q.Set("page", strconv.Itoa(opts.Page))
	}
	if opts.Size > 0 {
		q.Set("size", strconv.Itoa(opts.Size))
	}
	path := PathComputers
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	var out struct {
		Data  []Instance `json:"data"`
		Items []Instance `json:"items"`
	}
	if err := c.do(ctx, http.MethodGet, path, taoKey, nil, &out); err != nil {
		return nil, err
	}
	if out.Data != nil {
		return out.Data, nil
	}
	return out.Items, nil
}

// GetCatalog returns Tao catalog services.
func (c *Client) GetCatalog(ctx context.Context, taoKey string) (*Catalog, error) {
	var out Catalog
	if err := c.do(ctx, http.MethodGet, PathCatalog, taoKey, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) do(ctx context.Context, method, path, taoKey string, body, out any) error {
	return c.doWithIdempotency(ctx, method, path, taoKey, body, out, "")
}

func (c *Client) doWithIdempotency(ctx context.Context, method, path, taoKey string, body, out any, idempotencyKey string) error {
	if c == nil || c.Endpoint == "" {
		return fmt.Errorf("tao endpoint is not configured")
	}
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.Endpoint+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if taoKey != "" {
		req.Header.Set("Authorization", "Bearer "+taoKey)
	}
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{Status: resp.StatusCode, Body: data}
	}
	if out != nil && len(data) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("decode Tao response: %w", err)
		}
	}
	return nil
}

// APIError is a Tao HTTP error.
type APIError struct {
	Status int
	Body   []byte
}

func (e *APIError) Error() string {
	return fmt.Sprintf("tao api status %d: %s", e.Status, strings.TrimSpace(string(e.Body)))
}

var _ TaoComputersClient = (*Client)(nil)
