// Package taocloud provides an in-process mock of the Tao Cloud host-selling
// API. It is independent from Yao's Computer runtime and exists for Tao
// purchase/provisioning contract tests.
package taocloud

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Instance is one fake Tao Cloud host.
type Instance struct {
	InstanceID        string            `json:"instance_id"`
	Status            string            `json:"status"`
	RawStatus         string            `json:"raw_status,omitempty"`
	Region            string            `json:"region"`
	Resource          string            `json:"resource,omitempty"`
	BillingUnit       string            `json:"billing_unit,omitempty"`
	PeriodMonths      int               `json:"period_months,omitempty"`
	PriceCredits      int               `json:"price_credits,omitempty"`
	PriceTotalCredits *int              `json:"price_total_credits,omitempty"`
	ExpiresAt         *time.Time        `json:"expires_at,omitempty"`
	PublicIP          string            `json:"public_ip,omitempty"`
	PrivateIP         string            `json:"private_ip,omitempty"`
	SSH               map[string]any    `json:"ssh,omitempty"`
	Metadata          map[string]string `json:"metadata,omitempty"`
	LastError         string            `json:"last_error,omitempty"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
}

// DurationTier is a MONTH duration discount tier.
type DurationTier struct {
	Months   string  `json:"months"`
	Discount float64 `json:"discount"`
}

// Resource is one fake Tao Cloud catalog resource.
type Resource struct {
	ID             string         `json:"id"`
	Region         string         `json:"region"`
	CPU            int            `json:"cpu"`
	MemoryMB       int            `json:"memory_mb"`
	DiskGB         int            `json:"disk_gb"`
	BandwidthMbps  int            `json:"bandwidth_mbps"`
	BillingUnits   []string       `json:"billing_units"`
	Prices         map[string]int `json:"prices"`
	DurationTiers  []DurationTier `json:"duration_tiers,omitempty"`
	Images         []string       `json:"images"`
	Enabled        bool           `json:"enabled"`
	AvailableSlots int            `json:"available_slots"`
}

// RecordedRequest is retained for assertions.
type RecordedRequest struct {
	Method string
	Path   string
	Body   []byte
	Header http.Header
}

// Server is a Tao Cloud mock.
const (
	// DefaultTestAPIKey is the fixed Tao API key accepted by the mock.
	DefaultTestAPIKey = "sk-tao-e2e"
	// TestAPIKeyEnv overrides the mock key for UAT/local integration.
	TestAPIKeyEnv = "TAO_MOCK_API_KEY"
)

type Server struct {
	HTTP    *httptest.Server
	Token   string
	APIKeys []string
	AutoRun bool

	mu         sync.Mutex
	instances  map[string]*Instance
	resources  map[string]*Resource
	idempotent map[string]string
	requests   []RecordedRequest
	failures   []failure
	seq        uint64
}

type failure struct {
	method string
	path   string
	status int
	code   string
}

// New starts a Tao Cloud mock.
func New() *Server {
	token := os.Getenv(TestAPIKeyEnv)
	if token == "" {
		token = DefaultTestAPIKey
	}
	s := &Server{
		Token:      token,
		APIKeys:    []string{token},
		AutoRun:    true,
		instances:  map[string]*Instance{},
		resources:  map[string]*Resource{},
		idempotent: map[string]string{},
	}
	s.AddResource(Resource{ID: "starter-2c2g", Region: "bj-cn", CPU: 2, MemoryMB: 2048, DiskGB: 50, BandwidthMbps: 1, BillingUnits: []string{"MONTH"}, Prices: map[string]int{"MONTH": 12000}, Images: []string{"tao-base-1.0"}, Enabled: true, AvailableSlots: 100})
	s.AddResource(Resource{ID: "flex-2c2g", Region: "bj-cn", CPU: 2, MemoryMB: 2048, DiskGB: 50, BandwidthMbps: 1, BillingUnits: []string{"HOUR", "MONTH"}, Prices: map[string]int{"HOUR": 393, "MONTH": 286800}, DurationTiers: []DurationTier{{Months: "1-2", Discount: 0.87}, {Months: "3-5", Discount: 0.78}, {Months: "6-11", Discount: 0.73}, {Months: "12+", Discount: 0.64}}, Images: []string{"tao-base-1.0"}, Enabled: true, AvailableSlots: 100})
	s.HTTP = httptest.NewServer(http.HandlerFunc(s.handle))
	return s
}

// Handler exposes the mock HTTP handler for standalone UAT runs.
func (s *Server) Handler() http.Handler { return http.HandlerFunc(s.handle) }

// URL returns the mock base URL.
func (s *Server) URL() string { return s.HTTP.URL }

// Close stops the mock.
func (s *Server) Close() { s.HTTP.Close() }

// AddModel adds or replaces a catalog model.
func (s *Server) AddResource(p Resource) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(p.BillingUnits) == 0 {
		p.BillingUnits = []string{"HOUR"}
	}
	if p.Prices == nil {
		p.Prices = map[string]int{"HOUR": 393}
	}
	if p.Region == "" {
		p.Region = "bj-cn"
	}
	if len(p.Images) == 0 {
		p.Images = []string{"tao-base-1.0"}
	}
	p.Enabled = true
	s.resources[p.ID] = &p
}

// SetModelCapacity changes available capacity.
func (s *Server) SetResourceCapacity(resource string, slots int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p, ok := s.resources[resource]; ok {
		p.AvailableSlots = slots
	}
}

// SetInstanceStatus changes an instance state directly.
func (s *Server) SetInstanceStatus(id, status, lastError string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if instance, ok := s.instances[id]; ok {
		instance.Status = status
		instance.RawStatus = status
		instance.LastError = lastError
		instance.UpdatedAt = time.Now().UTC()
	}
}

// FailNext queues one error response for a method/path match.
func (s *Server) FailNext(method, pathContains string, status int, code string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failures = append(s.failures, failure{method: method, path: pathContains, status: status, code: code})
}

// Requests returns recorded requests.
func (s *Server) Requests() []RecordedRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]RecordedRequest, len(s.requests))
	copy(out, s.requests)
	return out
}

// GetInstance returns a copy of an instance.
func (s *Server) GetInstance(id string) (Instance, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	instance, ok := s.instances[id]
	if !ok {
		return Instance{}, false
	}
	return *instance, true
}

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	s.mu.Lock()
	s.requests = append(s.requests, RecordedRequest{Method: r.Method, Path: r.URL.Path, Body: body, Header: r.Header.Clone()})
	if f, ok := s.takeFailureLocked(r.Method, r.URL.Path); ok {
		s.mu.Unlock()
		if f.status == http.StatusTooManyRequests {
			w.Header().Set("Retry-After", "2")
		}
		writeError(w, f.status, f.code, "injected failure")
		return
	}
	s.mu.Unlock()

	if r.URL.Path == "/v1/health" {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "version": "test"})
		return
	}
	// Public catalog/service discovery endpoints (Tao-style, no API key).
	if r.Method == http.MethodGet && r.URL.Path == "/v1/catalog" {
		s.handleCatalog(w)
		return
	}
	if r.Method == http.MethodGet && r.URL.Path == "/v1/services" {
		s.handleServices(w)
		return
	}

	if !strings.HasPrefix(r.URL.Path, "/v1/computers") {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	if !s.authorized(r) {
		w.Header().Set("WWW-Authenticate", `Bearer realm="tao-mock"`)
		writeError(w, http.StatusUnauthorized, "unauthorized", "invalid Tao API key")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/v1/computers")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	switch {
	case r.Method == http.MethodGet && path == "/resources":
		s.handleResources(w)
	case r.Method == http.MethodGet && len(parts) == 3 && parts[0] == "resources" && parts[2] == "capacity":
		s.handleCapacity(w, parts[1], r.URL.Query().Get("region"), r.URL.Query().Get("billing_unit"))
	case r.Method == http.MethodPost && (path == "" || path == "/"):
		s.handleCreate(w, r, body)
	case r.Method == http.MethodGet && (path == "" || path == "/"):
		s.handleList(w, r)
	case r.Method == http.MethodGet && len(parts) == 1 && parts[0] != "":
		s.handleGet(w, parts[0])
	case r.Method == http.MethodPost && len(parts) == 2 && (parts[1] == "start" || parts[1] == "stop" || parts[1] == "reboot"):
		s.handleAction(w, parts[0], parts[1])
	case r.Method == http.MethodDelete && len(parts) == 1 && parts[0] != "":
		s.handleDelete(w, parts[0])
	default:
		writeError(w, http.StatusNotFound, "not_found", "not found")
	}
}

func (s *Server) takeFailureLocked(method, path string) (failure, bool) {
	for i, f := range s.failures {
		if f.method == method && strings.Contains(path, f.path) {
			s.failures = append(s.failures[:i], s.failures[i+1:]...)
			return f, true
		}
	}
	return failure{}, false
}

func (s *Server) authorized(r *http.Request) bool {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	for _, key := range s.APIKeys {
		if auth == "Bearer "+key {
			return true
		}
	}
	return false
}

// NewWithAPIKey starts a mock that accepts one explicit test key.
func NewWithAPIKey(key string) *Server {
	s := New()
	if key != "" {
		s.Token = key
		s.APIKeys = []string{key}
	}
	return s
}

// SetAPIKeys replaces the accepted test keys.
func (s *Server) SetAPIKeys(keys ...string) {
	if len(keys) == 0 {
		keys = []string{s.Token}
	}
	s.APIKeys = append([]string(nil), keys...)
}

func (s *Server) handleResources(w http.ResponseWriter) {
	s.mu.Lock()
	defer s.mu.Unlock()
	resources := make([]Resource, 0, len(s.resources))
	for _, res := range s.resources {
		resources = append(resources, *res)
	}
	writeJSON(w, http.StatusOK, map[string]any{"kind": "resource", "resources": resources})
}

func (s *Server) handleCapacity(w http.ResponseWriter, resource, region, billingUnit string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, ok := s.resources[resource]
	if !ok || !res.Enabled || (region != "" && res.Region != region) {
		writeError(w, http.StatusNotFound, "not_found", "resource not found")
		return
	}
	unit := billingUnit
	if unit == "" {
		if len(res.BillingUnits) != 1 {
			writeError(w, http.StatusBadRequest, "invalid_request", "billing_unit is required")
			return
		}
		unit = res.BillingUnits[0]
	}
	if !contains(res.BillingUnits, unit) {
		writeError(w, http.StatusBadRequest, "invalid_request", "unsupported billing_unit")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"resource": resource, "region": res.Region, "billing_unit": unit, "available_slots": res.AvailableSlots,
	})
}

func (s *Server) handleCreate(w http.ResponseWriter, r *http.Request, body []byte) {
	var req struct {
		Resource      string         `json:"resource"`
		Region        string         `json:"region"`
		Name          string         `json:"name"`
		BillingUnit   string         `json:"billing_unit"`
		PeriodMonths  int            `json:"period_months"`
		RequestParams map[string]any `json:"request_params"`
		SSH           *struct {
			PublicKey string `json:"public_key"`
			KeyID     string `json:"key_id"`
		} `json:"ssh"`
		Metadata map[string]string `json:"metadata"`
	}
	if err := json.Unmarshal(body, &req); err != nil || req.Resource == "" || req.Region == "" || req.Name == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "resource, region and name are required")
		return
	}
	if len(req.Name) > 50 {
		writeError(w, http.StatusBadRequest, "invalid_request", "name is too long")
		return
	}
	image, _ := req.RequestParams["image"].(string)
	if image == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "request_params.image is required")
		return
	}
	if req.SSH == nil || (req.SSH.PublicKey == "" && req.SSH.KeyID == "") {
		writeError(w, http.StatusBadRequest, "invalid_request", "ssh.public_key or ssh.key_id is required")
		return
	}
	if req.SSH.PublicKey != "" && !strings.HasPrefix(req.SSH.PublicKey, "ssh-") && !strings.HasPrefix(req.SSH.PublicKey, "ecdsa-sha2-") {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid ssh public key")
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	res, ok := s.resources[req.Resource]
	if !ok || !res.Enabled {
		writeError(w, http.StatusUnprocessableEntity, "unsupported_resource", "unsupported resource")
		return
	}
	if req.Region != res.Region {
		writeError(w, http.StatusBadRequest, "invalid_request", "resource does not support region")
		return
	}
	if !contains(res.Images, image) {
		writeError(w, http.StatusUnprocessableEntity, "unsupported_resource", "unsupported image")
		return
	}
	unit := req.BillingUnit
	if unit == "" {
		if len(res.BillingUnits) != 1 {
			writeError(w, http.StatusBadRequest, "invalid_request", "billing_unit is required")
			return
		}
		unit = res.BillingUnits[0]
	}
	if !contains(res.BillingUnits, unit) {
		writeError(w, http.StatusBadRequest, "invalid_request", "unsupported billing_unit")
		return
	}
	if unit == "MONTH" {
		if !containsInt([]int{1, 3, 6, 12}, req.PeriodMonths) {
			writeError(w, http.StatusBadRequest, "invalid_request", "period_months must be 1, 3, 6 or 12")
			return
		}
	} else if req.PeriodMonths != 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "period_months is only valid for MONTH")
		return
	}
	if res.AvailableSlots <= 0 {
		writeError(w, http.StatusConflict, "capacity_unavailable", "no capacity")
		return
	}
	idemKey := r.Header.Get("Idempotency-Key")
	if idemKey == "" {
		idemKey = req.Metadata["request_id"]
	}
	if idemKey != "" {
		if id, ok := s.idempotent[idemKey]; ok {
			if instance, ok := s.instances[id]; ok {
				w.Header().Set("Location", "/v1/computers/"+id)
				w.Header().Set("Retry-After", "2")
				writeJSON(w, http.StatusAccepted, instance)
				return
			}
		}
	}
	res.AvailableSlots--
	s.seq++
	id := fmt.Sprintf("inst-%d", s.seq)
	now := time.Now().UTC()
	price := res.Prices[unit]
	var total *int
	var expires *time.Time
	if unit == "MONTH" {
		value := price * req.PeriodMonths
		total = &value
		expiry := now.AddDate(0, req.PeriodMonths, 0)
		expires = &expiry
	}
	instance := &Instance{
		InstanceID: id, Status: "pending", RawStatus: "creating", Region: res.Region,
		Resource: res.ID, BillingUnit: unit, PeriodMonths: req.PeriodMonths,
		PriceCredits: price, PriceTotalCredits: total, ExpiresAt: expires,
		SSH:      map[string]any{"user": "root", "port": 22, "public_key_fingerprint": "SHA256:mock"},
		Metadata: req.Metadata, CreatedAt: now, UpdatedAt: now,
	}
	s.instances[id] = instance
	if idemKey != "" {
		s.idempotent[idemKey] = id
	}
	w.Header().Set("Location", "/v1/computers/"+id)
	w.Header().Set("Retry-After", "2")
	writeJSON(w, http.StatusAccepted, instance)
}

func (s *Server) handleGet(w http.ResponseWriter, id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	instance, ok := s.instances[id]
	if !ok || instance.Status == "deleted" {
		writeError(w, http.StatusNotFound, "not_found", "instance not found")
		return
	}
	if s.AutoRun && instance.Status == "pending" {
		instance.Status = "running"
		instance.RawStatus = "RUNNING"
		instance.PublicIP = "203.0.113.10"
		instance.PrivateIP = "10.0.0.10"
		instance.SSH["host"] = instance.PublicIP
		instance.UpdatedAt = time.Now().UTC()
	}
	writeJSON(w, http.StatusOK, instance)
}

func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	region := r.URL.Query().Get("region")
	page := 1
	size := 50
	if raw := r.URL.Query().Get("page"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid page")
			return
		}
		page = value
	}
	if raw := r.URL.Query().Get("size"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 100 {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid size")
			return
		}
		size = value
	}
	if status != "" && !contains([]string{"pending", "running", "stopped", "error", "deleted"}, status) {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid status")
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	instances := []Instance{}
	for _, instance := range s.instances {
		if instance.Status == "deleted" {
			continue
		}
		if status != "" && instance.Status != status {
			continue
		}
		if region != "" && instance.Region != region {
			continue
		}
		instances = append(instances, *instance)
	}
	total := len(instances)
	start := (page - 1) * size
	if start > total {
		start = total
	}
	end := start + size
	if end > total {
		end = total
	}
	writeJSON(w, http.StatusOK, map[string]any{"object": "list", "data": instances[start:end], "total": total, "page": page, "size": size})
}

func (s *Server) handleAction(w http.ResponseWriter, id, action string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	instance, ok := s.instances[id]
	if !ok || instance.Status == "deleted" {
		writeError(w, http.StatusNotFound, "not_found", "instance not found")
		return
	}
	switch action {
	case "start":
		if instance.Status == "running" {
			writeActionResponse(w, instance)
			return
		}
		if instance.Status != "stopped" {
			writeError(w, http.StatusConflict, "invalid_state", "start requires stopped")
			return
		}
		instance.Status = "running"
	case "stop":
		if instance.Status == "stopped" {
			writeActionResponse(w, instance)
			return
		}
		if instance.Status != "running" {
			writeError(w, http.StatusConflict, "invalid_state", "stop requires running")
			return
		}
		instance.Status = "stopped"
	case "reboot":
		if instance.Status != "running" {
			writeError(w, http.StatusConflict, "invalid_state", "reboot requires running")
			return
		}
		instance.Status = "pending"
	}
	instance.RawStatus = instance.Status
	instance.UpdatedAt = time.Now().UTC()
	writeActionResponse(w, instance)
}

func writeActionResponse(w http.ResponseWriter, instance *Instance) {
	w.Header().Set("Location", "/v1/computers/"+instance.InstanceID)
	w.Header().Set("Retry-After", "2")
	writeJSON(w, http.StatusAccepted, instance)
}

func (s *Server) handleDelete(w http.ResponseWriter, id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	instance, ok := s.instances[id]
	if !ok || instance.Status == "deleted" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	instance.Status = "deleted"
	instance.RawStatus = "deleted"
	instance.UpdatedAt = time.Now().UTC()
	w.Header().Set("Location", "/v1/computers/"+id)
	w.Header().Set("Retry-After", "2")
	writeJSON(w, http.StatusAccepted, instance)
}

func (s *Server) handleCatalog(w http.ResponseWriter) {
	s.mu.Lock()
	defer s.mu.Unlock()
	units := []string{}
	resources := make([]map[string]any, 0, len(s.resources))
	for _, res := range s.resources {
		for _, unit := range res.BillingUnits {
			if !contains(units, unit) {
				units = append(units, unit)
			}
		}
		resources = append(resources, map[string]any{
			"id": res.ID, "region": res.Region, "cpu": res.CPU, "memory_mb": res.MemoryMB,
			"disk_gb": res.DiskGB, "bandwidth_mbps": res.BandwidthMbps,
			"billing_units": res.BillingUnits, "prices": res.Prices,
			"duration_tiers": res.DurationTiers, "images": res.Images,
			"enabled": res.Enabled, "available_slots": res.AvailableSlots,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"object": "catalog", "locales": []string{"zh-cn"},
		"services": []map[string]any{{
			"id": "computer", "kind": "resource", "units": units,
			"endpoints": []map[string]any{
				{"id": "create", "method": "POST", "path": "/v1/computers", "unit": ""},
				{"id": "get", "method": "GET", "path": "/v1/computers/{id}", "unit": ""},
				{"id": "start", "method": "POST", "path": "/v1/computers/{id}/start", "unit": ""},
				{"id": "stop", "method": "POST", "path": "/v1/computers/{id}/stop", "unit": ""},
				{"id": "reboot", "method": "POST", "path": "/v1/computers/{id}/reboot", "unit": ""},
				{"id": "delete", "method": "DELETE", "path": "/v1/computers/{id}", "unit": ""},
			},
			"resources": resources,
		}},
	})
}

func (s *Server) handleServices(w http.ResponseWriter) {
	writeJSON(w, http.StatusOK, map[string]any{
		"object": "services",
		"data":   []map[string]any{{"id": "computer", "kind": "resource", "units": []string{"HOUR", "MONTH"}}},
	})
}

func contains(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}

func containsInt(values []int, value int) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}

func queryInt(r *http.Request, key string, def int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]any{"code": code, "message": message}})
}
