package taocloud_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/yaoapp/yao/e2e/taocloud"
)

func newClient(t *testing.T, srv *taocloud.Server) *http.Client {
	t.Helper()
	t.Cleanup(srv.Close)
	return srv.HTTP.Client()
}

func doJSON(t *testing.T, client *http.Client, method, url string, body any, token string) (int, []byte) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, data
}

func decodeInstance(t *testing.T, body []byte) taocloud.Instance {
	t.Helper()
	var instance taocloud.Instance
	if err := json.Unmarshal(body, &instance); err != nil {
		t.Fatalf("decode instance: %v body=%s", err, body)
	}
	return instance
}

func createBody(resource string) map[string]any {
	return map[string]any{
		"resource": resource, "region": "bj-cn", "name": "demo", "billing_unit": "HOUR",
		"request_params": map[string]any{"image": "tao-base-1.0"},
		"ssh":            map[string]any{"public_key": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI user@host"},
	}
}

func TestResourcesAndCatalog(t *testing.T) {
	srv := taocloud.New()
	client := newClient(t, srv)
	status, body := doJSON(t, client, http.MethodGet, srv.URL()+"/v1/computers/resources", nil, "")
	if status != http.StatusUnauthorized {
		t.Fatalf("anonymous resources status=%d", status)
	}
	status, body = doJSON(t, client, http.MethodGet, srv.URL()+"/v1/computers/resources", nil, srv.Token)
	if status != http.StatusOK || !bytes.Contains(body, []byte(`starter-2c2g`)) || !bytes.Contains(body, []byte(`flex-2c2g`)) || !bytes.Contains(body, []byte(`billing_units`)) {
		t.Fatalf("resources status=%d body=%s", status, body)
	}
	status, body = doJSON(t, client, http.MethodGet, srv.URL()+"/v1/computers/resources/flex-2c2g/capacity?region=bj-cn&billing_unit=HOUR", nil, srv.Token)
	if status != http.StatusOK || !bytes.Contains(body, []byte(`"billing_unit":"HOUR"`)) {
		t.Fatalf("capacity status=%d body=%s", status, body)
	}
	status, body = doJSON(t, client, http.MethodGet, srv.URL()+"/v1/catalog", nil, "")
	if status != http.StatusOK || !bytes.Contains(body, []byte(`"billing_units"`)) || !bytes.Contains(body, []byte(`"prices"`)) || !bytes.Contains(body, []byte(`"images"`)) {
		t.Fatalf("catalog status=%d body=%s", status, body)
	}
}

func TestCreateGetListLifecycleAndDeleted404(t *testing.T) {
	srv := taocloud.New()
	client := newClient(t, srv)
	status, body := doJSON(t, client, http.MethodPost, srv.URL()+"/v1/computers", createBody("flex-2c2g"), srv.Token)
	if status != http.StatusAccepted {
		t.Fatalf("create status=%d body=%s", status, body)
	}
	instance := decodeInstance(t, body)
	if instance.InstanceID == "" || instance.Status != "pending" || instance.Resource != "flex-2c2g" || instance.BillingUnit != "HOUR" {
		t.Fatalf("unexpected create: %+v", instance)
	}
	status, body = doJSON(t, client, http.MethodGet, srv.URL()+"/v1/computers/"+instance.InstanceID, nil, srv.Token)
	if status != http.StatusOK || decodeInstance(t, body).Status != "running" {
		t.Fatalf("get status=%d body=%s", status, body)
	}
	status, body = doJSON(t, client, http.MethodGet, srv.URL()+"/v1/computers?status=running&page=1&size=10", nil, srv.Token)
	if status != http.StatusOK || !bytes.Contains(body, []byte(`"object":"list"`)) || !bytes.Contains(body, []byte(instance.InstanceID)) {
		t.Fatalf("list status=%d body=%s", status, body)
	}
	status, body = doJSON(t, client, http.MethodPost, srv.URL()+"/v1/computers/"+instance.InstanceID+"/stop", nil, srv.Token)
	if status != http.StatusAccepted || decodeInstance(t, body).Status != "stopped" {
		t.Fatalf("stop status=%d body=%s", status, body)
	}
	status, body = doJSON(t, client, http.MethodPost, srv.URL()+"/v1/computers/"+instance.InstanceID+"/stop", nil, srv.Token)
	if status != http.StatusAccepted {
		t.Fatalf("idempotent stop status=%d body=%s", status, body)
	}
	status, _ = doJSON(t, client, http.MethodDelete, srv.URL()+"/v1/computers/"+instance.InstanceID, nil, srv.Token)
	if status != http.StatusAccepted {
		t.Fatalf("delete status=%d", status)
	}
	status, _ = doJSON(t, client, http.MethodDelete, srv.URL()+"/v1/computers/"+instance.InstanceID, nil, srv.Token)
	if status != http.StatusNoContent {
		t.Fatalf("repeat delete status=%d", status)
	}
	status, _ = doJSON(t, client, http.MethodGet, srv.URL()+"/v1/computers/"+instance.InstanceID, nil, srv.Token)
	if status != http.StatusNotFound {
		t.Fatalf("deleted GET status=%d", status)
	}
}

func TestMonthAndIdempotency(t *testing.T) {
	srv := taocloud.New()
	client := newClient(t, srv)
	body := map[string]any{
		"resource": "starter-2c2g", "region": "bj-cn", "name": "month", "billing_unit": "MONTH", "period_months": 3,
		"request_params": map[string]any{"image": "tao-base-1.0"},
		"ssh":            map[string]any{"public_key": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI user@host"},
	}
	status, data := doJSON(t, client, http.MethodPost, srv.URL()+"/v1/computers", body, srv.Token)
	if status != http.StatusAccepted {
		t.Fatalf("month create status=%d body=%s", status, data)
	}
	instance := decodeInstance(t, data)
	if instance.BillingUnit != "MONTH" || instance.PeriodMonths != 3 || instance.PriceTotalCredits == nil || instance.ExpiresAt == nil {
		t.Fatalf("month instance=%+v", instance)
	}
	// Idempotency-Key must return the same instance.
	req, _ := http.NewRequest(http.MethodPost, srv.URL()+"/v1/computers", bytes.NewReader(mustJSON(t, createBody("flex-2c2g"))))
	req.Header.Set("Authorization", "Bearer "+srv.Token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "idem-1")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	var first taocloud.Instance
	_ = json.NewDecoder(resp.Body).Decode(&first)
	resp.Body.Close()
	req2, _ := http.NewRequest(http.MethodPost, srv.URL()+"/v1/computers", bytes.NewReader(mustJSON(t, createBody("flex-2c2g"))))
	req2.Header = req.Header.Clone()
	resp2, err := client.Do(req2)
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	var second taocloud.Instance
	_ = json.NewDecoder(resp2.Body).Decode(&second)
	if resp2.StatusCode != http.StatusAccepted || second.InstanceID == "" || second.InstanceID != first.InstanceID {
		t.Fatalf("idem status=%d first=%s second=%s", resp2.StatusCode, first.InstanceID, second.InstanceID)
	}
}

func TestValidationAndErrors(t *testing.T) {
	srv := taocloud.New()
	client := newClient(t, srv)
	cases := []struct {
		name string
		body map[string]any
		want int
	}{
		{"missing resource", map[string]any{}, http.StatusBadRequest},
		{"missing image", map[string]any{"resource": "flex-2c2g", "region": "bj-cn", "name": "x", "billing_unit": "HOUR", "ssh": map[string]any{"public_key": "ssh-ed25519 A"}}, http.StatusBadRequest},
		{"missing ssh", map[string]any{"resource": "flex-2c2g", "region": "bj-cn", "name": "x", "billing_unit": "HOUR", "request_params": map[string]any{"image": "tao-base-1.0"}}, http.StatusBadRequest},
		{"unsupported unit", map[string]any{"resource": "starter-2c2g", "region": "bj-cn", "name": "x", "billing_unit": "HOUR", "period_months": 1, "request_params": map[string]any{"image": "tao-base-1.0"}, "ssh": map[string]any{"public_key": "ssh-ed25519 A"}}, http.StatusBadRequest},
		{"unknown resource", createBody("missing"), http.StatusUnprocessableEntity},
	}
	for _, tc := range cases {
		status, body := doJSON(t, client, http.MethodPost, srv.URL()+"/v1/computers", tc.body, srv.Token)
		if status != tc.want {
			t.Fatalf("%s status=%d body=%s", tc.name, status, body)
		}
	}
	status, body := doJSON(t, client, http.MethodGet, srv.URL()+"/v1/computers?size=101", nil, srv.Token)
	if status != http.StatusBadRequest {
		t.Fatalf("pagination status=%d body=%s", status, body)
	}
	status, body = doJSON(t, client, http.MethodGet, srv.URL()+"/v1/computers?status=weird", nil, srv.Token)
	if status != http.StatusBadRequest {
		t.Fatalf("status filter status=%d body=%s", status, body)
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestInjectedErrorCodes(t *testing.T) {
	srv := taocloud.New()
	client := newClient(t, srv)
	cases := []struct {
		status int
		code   string
	}{
		{http.StatusPaymentRequired, "insufficient_credit"},
		{http.StatusForbidden, "forbidden"},
		{http.StatusTooManyRequests, "rate_limited"},
		{http.StatusServiceUnavailable, "cloud_unavailable"},
		{http.StatusInternalServerError, "internal_error"},
	}
	for _, tc := range cases {
		srv.FailNext(http.MethodPost, "/v1/computers", tc.status, tc.code)
		status, body := doJSON(t, client, http.MethodPost, srv.URL()+"/v1/computers", createBody("flex-2c2g"), srv.Token)
		if status != tc.status || !bytes.Contains(body, []byte(tc.code)) {
			t.Fatalf("%s status=%d body=%s", tc.code, status, body)
		}
	}
}

func TestMockTestAPIKey(t *testing.T) {
	srv := taocloud.New()
	client := newClient(t, srv)
	status, body := doJSON(t, client, http.MethodGet, srv.URL()+"/v1/computers/resources", nil, taocloud.DefaultTestAPIKey)
	if status != http.StatusOK {
		t.Fatalf("default test key status=%d body=%s", status, body)
	}
	status, _ = doJSON(t, client, http.MethodGet, srv.URL()+"/v1/computers/resources", nil, "sk-tao-wrong")
	if status != http.StatusUnauthorized {
		t.Fatalf("wrong key status=%d", status)
	}
}

func TestMockAPIKeyEnvOverride(t *testing.T) {
	t.Setenv(taocloud.TestAPIKeyEnv, "sk-tao-uat")
	srv := taocloud.New()
	client := newClient(t, srv)
	status, body := doJSON(t, client, http.MethodGet, srv.URL()+"/v1/computers/resources", nil, "sk-tao-uat")
	if status != http.StatusOK {
		t.Fatalf("uat key status=%d body=%s", status, body)
	}
	status, _ = doJSON(t, client, http.MethodGet, srv.URL()+"/v1/computers/resources", nil, taocloud.DefaultTestAPIKey)
	if status != http.StatusUnauthorized {
		t.Fatalf("default key should be rejected, status=%d", status)
	}
}

func TestMockSetAPIKeys(t *testing.T) {
	srv := taocloud.New()
	srv.SetAPIKeys("sk-tao-a", "sk-tao-b")
	client := newClient(t, srv)
	for _, key := range []string{"sk-tao-a", "sk-tao-b"} {
		status, body := doJSON(t, client, http.MethodGet, srv.URL()+"/v1/computers/resources", nil, key)
		if status != http.StatusOK {
			t.Fatalf("key %s status=%d body=%s", key, status, body)
		}
	}
}
