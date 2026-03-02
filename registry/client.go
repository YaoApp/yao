// Package registry provides a client SDK for the Yao Registry HTTP API.
// It supports push, pull, search, version management, dist-tags,
// dependency queries, and package deletion with Basic Auth.
package registry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Client talks to a Yao Registry server over HTTP.
// On first API call it discovers the API prefix via /.well-known/yao-registry
// so callers only need to provide the server root URL (e.g. "https://registry.yaoagents.com").
type Client struct {
	baseURL      string
	apiPrefix    string // resolved from well-known, e.g. "/v1"
	discoverOnce sync.Once
	username     string
	password     string
	httpClient   *http.Client
}

// Option configures a Client.
type Option func(*Client)

// WithAuth sets Basic Auth credentials for push/delete operations.
func WithAuth(username, password string) Option {
	return func(c *Client) {
		c.username = username
		c.password = password
	}
}

// WithHTTPClient overrides the default http.Client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.httpClient = hc }
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) { c.httpClient.Timeout = d }
}

// New creates a registry client. serverURL is the root URL users configure,
// e.g. "http://localhost:8080" or "https://registry.yaoagents.com".
// The actual API prefix is auto-discovered via /.well-known/yao-registry.
func New(serverURL string, opts ...Option) *Client {
	c := &Client{
		baseURL:    strings.TrimRight(serverURL, "/"),
		apiPrefix:  "/v1", // sensible default, overridden by discovery
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// ensureDiscovered runs well-known discovery exactly once (thread-safe).
func (c *Client) ensureDiscovered() {
	c.discoverOnce.Do(func() {
		var info RegistryInfo
		if err := c.doGet("/.well-known/yao-registry", nil, &info); err == nil && info.Registry.API != "" {
			c.apiPrefix = strings.TrimRight(info.Registry.API, "/")
		}
	})
}

// --- Response types ---

// RegistryInfo is returned by the discovery endpoint.
type RegistryInfo struct {
	Registry struct {
		Version string `json:"version"`
		API     string `json:"api"`
	} `json:"registry"`
	Types []string `json:"types"`
}

// ServerInfo is returned by GET /v1/.
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// PushResult is returned after a successful push.
type PushResult struct {
	Type    string `json:"type"`
	Scope   string `json:"scope"`
	Name    string `json:"name"`
	Version string `json:"version"`
	Digest  string `json:"digest"`
}

// DeleteResult is returned after a successful version delete.
type DeleteResult struct {
	Deleted string `json:"deleted"`
	Type    string `json:"type"`
	Scope   string `json:"scope"`
	Name    string `json:"name"`
}

// TagResult is returned after setting a dist-tag.
type TagResult struct {
	Tag     string `json:"tag"`
	Version string `json:"version"`
}

// TagDeleteResult is returned after deleting a dist-tag.
type TagDeleteResult struct {
	Deleted string `json:"deleted"`
}

// ListResult is returned by list and search endpoints.
type ListResult struct {
	Total    int               `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"pagesize"`
	Packages []json.RawMessage `json:"packages"`
}

// Packument is the full package metadata response.
type Packument struct {
	Type        string                     `json:"type"`
	Scope       string                     `json:"scope"`
	Name        string                     `json:"name"`
	Description string                     `json:"description"`
	Keywords    []string                   `json:"keywords"`
	DistTags    map[string]string          `json:"dist_tags"`
	Versions    map[string]json.RawMessage `json:"versions"`
	License     string                     `json:"license,omitempty"`
	Homepage    string                     `json:"homepage,omitempty"`
	Readme      string                     `json:"readme,omitempty"`
	Author      json.RawMessage            `json:"author,omitempty"`
	Maintainers json.RawMessage            `json:"maintainers,omitempty"`
	Repository  json.RawMessage            `json:"repository,omitempty"`
	Bugs        json.RawMessage            `json:"bugs,omitempty"`
	CreatedAt   string                     `json:"created_at"`
	UpdatedAt   string                     `json:"updated_at"`
}

// VersionDetail is returned for a single version query.
type VersionDetail struct {
	Type         string                 `json:"type"`
	Scope        string                 `json:"scope"`
	Name         string                 `json:"name"`
	Version      string                 `json:"version"`
	Digest       string                 `json:"digest"`
	Size         int64                  `json:"size"`
	Dependencies []Dependency           `json:"dependencies"`
	Metadata     map[string]interface{} `json:"metadata"`
	CreatedAt    string                 `json:"created_at"`
	Artifacts    []Artifact             `json:"artifacts,omitempty"`
}

// Dependency represents a package dependency.
type Dependency struct {
	Type    string `json:"type"`
	Scope   string `json:"scope"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

// DependencyList wraps the dependencies response.
type DependencyList struct {
	Dependencies []json.RawMessage `json:"dependencies"`
}

// DependentList wraps the dependents response.
type DependentList struct {
	Dependents []json.RawMessage `json:"dependents"`
}

// Artifact represents a platform-specific release artifact.
type Artifact struct {
	OS      string `json:"os"`
	Arch    string `json:"arch"`
	Variant string `json:"variant"`
	Digest  string `json:"digest"`
	Size    int64  `json:"size"`
}

// APIError is returned when the server responds with an error.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("registry: HTTP %d: %s", e.StatusCode, e.Message)
}

// --- Discovery ---

// Discover calls GET /.well-known/yao-registry.
func (c *Client) Discover() (*RegistryInfo, error) {
	var info RegistryInfo
	if err := c.doGet("/.well-known/yao-registry", nil, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// Info calls GET {apiPrefix}/.
func (c *Client) Info() (*ServerInfo, error) {
	c.ensureDiscovered()
	var info ServerInfo
	if err := c.doGet(c.apiPrefix+"/", nil, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// --- List & Search ---

// List calls GET {apiPrefix}/:type with optional filters.
func (c *Client) List(pkgType string, scope string, query string, page, pageSize int) (*ListResult, error) {
	c.ensureDiscovered()
	params := url.Values{}
	if scope != "" {
		params.Set("scope", scope)
	}
	if query != "" {
		params.Set("q", query)
	}
	if page > 0 {
		params.Set("page", fmt.Sprintf("%d", page))
	}
	if pageSize > 0 {
		params.Set("pagesize", fmt.Sprintf("%d", pageSize))
	}
	var result ListResult
	if err := c.doGet(c.apiPrefix+"/"+pkgType, params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Search calls GET {apiPrefix}/search.
func (c *Client) Search(q string, pkgType string, page, pageSize int) (*ListResult, error) {
	c.ensureDiscovered()
	params := url.Values{"q": {q}}
	if pkgType != "" {
		params.Set("type", pkgType)
	}
	if page > 0 {
		params.Set("page", fmt.Sprintf("%d", page))
	}
	if pageSize > 0 {
		params.Set("pagesize", fmt.Sprintf("%d", pageSize))
	}
	var result ListResult
	if err := c.doGet(c.apiPrefix+"/search", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// --- Package metadata ---

// GetPackument calls GET {apiPrefix}/:type/:scope/:name.
func (c *Client) GetPackument(pkgType, scope, name string) (*Packument, error) {
	c.ensureDiscovered()
	var p Packument
	path := fmt.Sprintf("%s/%s/%s/%s", c.apiPrefix, pkgType, scope, name)
	if err := c.doGet(path, nil, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// GetVersion calls GET {apiPrefix}/:type/:scope/:name/:version.
func (c *Client) GetVersion(pkgType, scope, name, version string) (*VersionDetail, error) {
	c.ensureDiscovered()
	var v VersionDetail
	path := fmt.Sprintf("%s/%s/%s/%s/%s", c.apiPrefix, pkgType, scope, name, version)
	if err := c.doGet(path, nil, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// --- Dependencies ---

// GetDependencies calls GET {apiPrefix}/:type/:scope/:name/:version/dependencies.
func (c *Client) GetDependencies(pkgType, scope, name, version string, recursive bool) (*DependencyList, error) {
	c.ensureDiscovered()
	path := fmt.Sprintf("%s/%s/%s/%s/%s/dependencies", c.apiPrefix, pkgType, scope, name, version)
	params := url.Values{}
	if recursive {
		params.Set("recursive", "true")
	}
	var dl DependencyList
	if err := c.doGet(path, params, &dl); err != nil {
		return nil, err
	}
	return &dl, nil
}

// GetDependents calls GET {apiPrefix}/:type/:scope/:name/dependents.
func (c *Client) GetDependents(pkgType, scope, name string) (*DependentList, error) {
	c.ensureDiscovered()
	path := fmt.Sprintf("%s/%s/%s/%s/dependents", c.apiPrefix, pkgType, scope, name)
	var dl DependentList
	if err := c.doGet(path, nil, &dl); err != nil {
		return nil, err
	}
	return &dl, nil
}

// --- Push & Pull ---

// Push uploads a .yao.zip package via PUT {apiPrefix}/:type/:scope/:name/:version.
func (c *Client) Push(pkgType, scope, name, version string, zipData []byte) (*PushResult, error) {
	c.ensureDiscovered()
	path := fmt.Sprintf("%s/%s/%s/%s/%s", c.apiPrefix, pkgType, scope, name, version)
	req, err := http.NewRequest(http.MethodPut, c.baseURL+path, bytes.NewReader(zipData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/zip")
	c.setAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, parseError(resp)
	}

	var result PushResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Pull downloads a .yao.zip via GET {apiPrefix}/:type/:scope/:name/:version/pull.
// The version parameter can be a semver or a dist-tag name.
func (c *Client) Pull(pkgType, scope, name, version string) ([]byte, string, error) {
	c.ensureDiscovered()
	path := fmt.Sprintf("%s/%s/%s/%s/%s/pull", c.apiPrefix, pkgType, scope, name, version)
	resp, err := c.httpClient.Get(c.baseURL + path)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", parseError(resp)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	digest := resp.Header.Get("X-Digest")
	return data, digest, nil
}

// --- Tags ---

// SetTag calls PUT {apiPrefix}/:type/:scope/:name/tags/:tag.
func (c *Client) SetTag(pkgType, scope, name, tag, version string) (*TagResult, error) {
	c.ensureDiscovered()
	path := fmt.Sprintf("%s/%s/%s/%s/tags/%s", c.apiPrefix, pkgType, scope, name, tag)
	body, _ := json.Marshal(map[string]string{"version": version})

	req, err := http.NewRequest(http.MethodPut, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	c.setAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp)
	}

	var result TagResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteTag calls DELETE {apiPrefix}/:type/:scope/:name/tags/:tag.
func (c *Client) DeleteTag(pkgType, scope, name, tag string) (*TagDeleteResult, error) {
	c.ensureDiscovered()
	path := fmt.Sprintf("%s/%s/%s/%s/tags/%s", c.apiPrefix, pkgType, scope, name, tag)
	req, err := http.NewRequest(http.MethodDelete, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	c.setAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp)
	}

	var result TagDeleteResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// --- Delete ---

// DeleteVersion calls DELETE {apiPrefix}/:type/:scope/:name/:version.
func (c *Client) DeleteVersion(pkgType, scope, name, version string) (*DeleteResult, error) {
	c.ensureDiscovered()
	path := fmt.Sprintf("%s/%s/%s/%s/%s", c.apiPrefix, pkgType, scope, name, version)
	req, err := http.NewRequest(http.MethodDelete, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	c.setAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp)
	}

	var result DeleteResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// --- Internal helpers ---

func (c *Client) setAuth(req *http.Request) {
	if c.username != "" {
		req.SetBasicAuth(c.username, c.password)
	}
}

func (c *Client) doGet(path string, params url.Values, out interface{}) error {
	u := c.baseURL + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	resp, err := c.httpClient.Get(u)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return parseError(resp)
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

func parseError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	var errResp struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(body, &errResp) == nil && errResp.Error != "" {
		return &APIError{StatusCode: resp.StatusCode, Message: errResp.Error}
	}
	return &APIError{StatusCode: resp.StatusCode, Message: string(body)}
}
