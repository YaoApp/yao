package setting

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// ---------------------------------------------------------------------------
// typesafeValidateKey
// ---------------------------------------------------------------------------

func TestTypesafeValidateKey_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/v1/models" {
			t.Fatalf("path = %s, want /v1/models", r.URL.Path)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-key" {
			t.Fatalf("auth = %q, want Bearer test-key", auth)
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": "jev-latest"},
			},
		})
	}))
	defer srv.Close()

	err := typesafeValidateKey(srv.URL, "test-key")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestTypesafeValidateKey_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	err := typesafeValidateKey(srv.URL, "bad-key")
	if err == nil {
		t.Fatal("expected error for 401")
	}
}

func TestTypesafeValidateKey_Forbidden(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	err := typesafeValidateKey(srv.URL, "bad-key")
	if err == nil {
		t.Fatal("expected error for 403")
	}
}

func TestTypesafeValidateKey_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	err := typesafeValidateKey(srv.URL, "key")
	if err == nil {
		t.Fatal("expected error for 500")
	}
}

func TestTypesafeValidateKey_ConnectionRefused(t *testing.T) {
	err := typesafeValidateKey("http://127.0.0.1:1", "key")
	if err == nil {
		t.Fatal("expected error for connection refused")
	}
}

func TestTypesafeValidateKey_TrailingSlash(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("path = %s, want /v1/models (trailing slash stripped)", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := typesafeValidateKey(srv.URL+"/", "key")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// fetchDecisionModels
// ---------------------------------------------------------------------------

func TestFetchDecisionModels_Empty(t *testing.T) {
	remoteModelCacheMu.Lock()
	remoteModelRawCache = nil
	remoteModelCacheMu.Unlock()

	models := fetchDecisionModels("http://unused", "key")
	if models != nil {
		t.Fatalf("expected nil when raw cache is nil, got %v", models)
	}
}

func TestFetchDecisionModels_NoDecisionService(t *testing.T) {
	remoteModelCacheMu.Lock()
	remoteModelRawCache = []map[string]interface{}{
		{"id": "gpt-4o", "service": "llm", "name": "GPT-4o"},
		{"id": "whisper-1", "service": "audio", "name": "Whisper"},
	}
	remoteModelCacheMu.Unlock()

	models := fetchDecisionModels("http://unused", "key")
	if len(models) != 0 {
		t.Fatalf("expected 0 decision models, got %d", len(models))
	}
}

func TestFetchDecisionModels_ExtractsDecisionModels(t *testing.T) {
	remoteModelCacheMu.Lock()
	remoteModelRawCache = []map[string]interface{}{
		{"id": "gpt-4o", "service": "llm", "name": "GPT-4o"},
		{"id": "jev-latest", "service": "decision", "name": "Jev Latest", "max_input_tokens": float64(32000)},
		{"id": "jev-1.13.0", "service": "decision", "name": "Jev 1.13"},
		{"id": "whisper-1", "service": "audio", "name": "Whisper"},
	}
	remoteModelCacheMu.Unlock()

	models := fetchDecisionModels("http://unused", "key")
	if len(models) != 2 {
		t.Fatalf("expected 2 decision models, got %d", len(models))
	}

	m0 := models[0]
	if m0.ID != "jev-latest" {
		t.Errorf("models[0].ID = %q, want jev-latest", m0.ID)
	}
	if m0.Name != "Jev Latest" {
		t.Errorf("models[0].Name = %q, want Jev Latest", m0.Name)
	}
	if m0.MaxInputTokens != 32000 {
		t.Errorf("models[0].MaxInputTokens = %d, want 32000", m0.MaxInputTokens)
	}
	if !m0.Enabled {
		t.Error("models[0] should be enabled")
	}
	found := false
	for _, c := range m0.Capabilities {
		if c == "decision" {
			found = true
		}
	}
	if !found {
		t.Errorf("models[0].Capabilities = %v, want [decision]", m0.Capabilities)
	}

	m1 := models[1]
	if m1.ID != "jev-1.13.0" {
		t.Errorf("models[1].ID = %q, want jev-1.13.0", m1.ID)
	}
}

func TestFetchDecisionModels_SkipsEmptyID(t *testing.T) {
	remoteModelCacheMu.Lock()
	remoteModelRawCache = []map[string]interface{}{
		{"id": "", "service": "decision", "name": "Bad Entry"},
		{"service": "decision", "name": "No ID Field"},
		{"id": "jev-latest", "service": "decision", "name": "Good"},
	}
	remoteModelCacheMu.Unlock()

	models := fetchDecisionModels("http://unused", "key")
	if len(models) != 1 {
		t.Fatalf("expected 1 model (skipping empty IDs), got %d", len(models))
	}
	if models[0].ID != "jev-latest" {
		t.Errorf("expected jev-latest, got %q", models[0].ID)
	}
}

func TestFetchDecisionModels_NameFallsBackToID(t *testing.T) {
	remoteModelCacheMu.Lock()
	remoteModelRawCache = []map[string]interface{}{
		{"id": "jev-latest", "service": "decision"},
	}
	remoteModelCacheMu.Unlock()

	models := fetchDecisionModels("http://unused", "key")
	if len(models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(models))
	}
	if models[0].Name != "jev-latest" {
		t.Errorf("Name should fall back to ID, got %q", models[0].Name)
	}
}

// ---------------------------------------------------------------------------
// mapRemoteModel — decision filtering
// ---------------------------------------------------------------------------

func TestMapRemoteModel_FiltersDecision(t *testing.T) {
	item := map[string]interface{}{
		"id":      "jev-latest",
		"service": "decision",
		"name":    "Jev Latest",
	}
	result := mapRemoteModel(item)
	if result != nil {
		t.Fatal("decision models should be filtered out by mapRemoteModel")
	}
}

func TestMapRemoteModel_PassesLLM(t *testing.T) {
	item := map[string]interface{}{
		"id":           "gpt-4o",
		"name":         "GPT-4o",
		"capabilities": []interface{}{"chat", "streaming"},
	}
	result := mapRemoteModel(item)
	if result == nil {
		t.Fatal("LLM model should pass mapRemoteModel")
	}
	if result["id"] != "gpt-4o" {
		t.Errorf("id = %v, want gpt-4o", result["id"])
	}
}

// ---------------------------------------------------------------------------
// typesafeValidateKey — real API (requires TYPESAFE_API_KEY)
// ---------------------------------------------------------------------------

func TestTypesafeValidateKey_RealAPI(t *testing.T) {
	key := os.Getenv("TYPESAFE_API_KEY")
	if key == "" {
		t.Fatal("TYPESAFE_API_KEY not set — add it to agent-test.env or export before running")
	}

	err := typesafeValidateKey("https://api.typesafe.ai", key)
	if err != nil {
		t.Fatalf("real API validation failed: %v", err)
	}
}

func TestTypesafeValidateKey_RealAPI_BadKey(t *testing.T) {
	key := os.Getenv("TYPESAFE_API_KEY")
	if key == "" {
		t.Fatal("TYPESAFE_API_KEY not set — add it to agent-test.env or export before running")
	}

	err := typesafeValidateKey("https://api.typesafe.ai", "invalid-key-xxx")
	if err == nil {
		t.Fatal("expected error for invalid key against real API")
	}
}
