package setting

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/yaoapp/yao/llmprovider"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/setting"
)

// ---------------------------------------------------------------------------
// resolveTaoBaseURL
// ---------------------------------------------------------------------------

func TestResolveTaoBaseURL_Defaults(t *testing.T) {
	os.Unsetenv("TAO_BASE_URL_CN")
	os.Unsetenv("TAO_BASE_URL_EN")

	got := resolveTaoBaseURL("zh-CN")
	if got != defaultTaoBaseURLCN {
		t.Fatalf("zh-CN: want %s, got %s", defaultTaoBaseURLCN, got)
	}

	got = resolveTaoBaseURL("en-us")
	if got != defaultTaoBaseURLEN {
		t.Fatalf("en-us: want %s, got %s", defaultTaoBaseURLEN, got)
	}

	got = resolveTaoBaseURL("")
	if got != defaultTaoBaseURLEN {
		t.Fatalf("empty locale: want %s, got %s", defaultTaoBaseURLEN, got)
	}
}

func TestResolveTaoBaseURL_EnvOverride(t *testing.T) {
	os.Setenv("TAO_BASE_URL_CN", "https://test-cn.example.com/")
	os.Setenv("TAO_BASE_URL_EN", "https://test-en.example.com/")
	defer os.Unsetenv("TAO_BASE_URL_CN")
	defer os.Unsetenv("TAO_BASE_URL_EN")

	got := resolveTaoBaseURL("zh-cn")
	if got != "https://test-cn.example.com" {
		t.Fatalf("zh-cn override: want https://test-cn.example.com, got %s", got)
	}

	got = resolveTaoBaseURL("en-US")
	if got != "https://test-en.example.com" {
		t.Fatalf("en-US override: want https://test-en.example.com, got %s", got)
	}
}

// ---------------------------------------------------------------------------
// taoCallVerify
// ---------------------------------------------------------------------------

func TestTaoCallVerify_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/key" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key-123" {
			t.Fatalf("unexpected auth: %s", r.Header.Get("Authorization"))
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"valid":             true,
			"balance":           99500,
			"balance_available": true,
		})
	}))
	defer srv.Close()

	result := taoCallVerify(srv.URL, "test-key-123")
	if !result.Valid {
		t.Fatal("expected valid=true")
	}
	if result.Balance == nil || *result.Balance != 99500 {
		t.Fatalf("expected balance=99500, got %v", result.Balance)
	}
	if !result.BalanceAvailable {
		t.Fatal("expected balance_available=true")
	}
}

func TestTaoCallVerify_InvalidKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"type":    "invalid_api_key",
				"message": "The API key is invalid",
			},
		})
	}))
	defer srv.Close()

	result := taoCallVerify(srv.URL, "bad-key")
	if result.Valid {
		t.Fatal("expected valid=false")
	}
	if result.ErrorType != "invalid_api_key" {
		t.Fatalf("expected error_type=invalid_api_key, got %s", result.ErrorType)
	}
}

func TestTaoCallVerify_ExpiredKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"type":    "api_key_expired",
				"message": "The API key has expired",
			},
		})
	}))
	defer srv.Close()

	result := taoCallVerify(srv.URL, "expired-key")
	if result.Valid {
		t.Fatal("expected valid=false")
	}
	if result.ErrorType != "api_key_expired" {
		t.Fatalf("expected error_type=api_key_expired, got %s", result.ErrorType)
	}
}

// ---------------------------------------------------------------------------
// maskKey
// ---------------------------------------------------------------------------

func TestMaskKey(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"abc", "***"},
		{"abcd", "****"},
		{"sk-tao-abcdefgh1234", "sk-...1234"},
	}
	for _, tc := range tests {
		got := maskKey(tc.in)
		if got != tc.want {
			t.Errorf("maskKey(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// taoFetchBalance
// ---------------------------------------------------------------------------

func TestTaoFetchBalance_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"valid":             true,
			"balance":           50000,
			"balance_available": true,
		})
	}))
	defer srv.Close()

	bal, avail := taoFetchBalance(srv.URL, "key", 5*time.Second)
	if !avail {
		t.Fatal("expected balance_available=true")
	}
	if bal == nil || *bal != 50000 {
		t.Fatalf("expected balance=50000, got %v", bal)
	}
}

func TestTaoFetchBalance_Failure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	bal, avail := taoFetchBalance(srv.URL, "key", 5*time.Second)
	if avail {
		t.Fatal("expected balance_available=false on error")
	}
	if bal != nil {
		t.Fatalf("expected nil balance on error, got %v", bal)
	}
}

// ---------------------------------------------------------------------------
// taoFetchModelDefaults
// ---------------------------------------------------------------------------

func TestTaoFetchModelDefaults_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/model-defaults" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "" {
			t.Fatal("model-defaults should not send auth header")
		}
		// Real Tao API returns mixed format: objects for chat models, strings for audio/embedding
		json.NewEncoder(w).Encode(map[string]interface{}{
			"defaults": map[string]interface{}{
				"default": map[string]interface{}{
					"model":            "deepseek-flash",
					"thinking":         map[string]interface{}{"type": "enabled"},
					"reasoning_effort": "high",
				},
				"heavy": map[string]interface{}{
					"model":            "deepseek-flash",
					"reasoning_effort": "max",
				},
				"light": map[string]interface{}{
					"model":    "deepseek-flash",
					"thinking": map[string]interface{}{"type": "disabled"},
				},
				"audio":     "whisper-1",
				"embedding": "text-embedding-3-small",
			},
		})
	}))
	defer srv.Close()

	roles := taoFetchModelDefaults(srv.URL)
	if roles == nil {
		t.Fatal("expected non-nil roles")
	}
	if roles["default"] == nil || roles["default"].Model != "deepseek-flash" {
		t.Fatalf("expected default model=deepseek-flash, got %v", roles["default"])
	}
	if roles["heavy"] == nil || roles["heavy"].Model != "deepseek-flash" {
		t.Fatalf("expected heavy model=deepseek-flash, got %v", roles["heavy"])
	}
	if roles["light"] == nil || roles["light"].Model != "deepseek-flash" {
		t.Fatalf("expected light model=deepseek-flash, got %v", roles["light"])
	}
	// String-format roles (audio, embedding) must be parsed correctly
	if roles["audio"] == nil || roles["audio"].Model != "whisper-1" {
		t.Fatalf("expected audio model=whisper-1, got %v", roles["audio"])
	}
	if roles["embedding"] == nil || roles["embedding"].Model != "text-embedding-3-small" {
		t.Fatalf("expected embedding model=text-embedding-3-small, got %v", roles["embedding"])
	}
	if re, _ := roles["default"].Params["reasoning_effort"].(string); re != "high" {
		t.Fatalf("expected default reasoning_effort=high, got %s", re)
	}
	if re, _ := roles["heavy"].Params["reasoning_effort"].(string); re != "max" {
		t.Fatalf("expected heavy reasoning_effort=max, got %s", re)
	}
	// String-format roles have empty params
	if len(roles["audio"].Params) != 0 {
		t.Fatalf("expected audio params to be empty, got %v", roles["audio"].Params)
	}
	if len(roles["embedding"].Params) != 0 {
		t.Fatalf("expected embedding params to be empty, got %v", roles["embedding"].Params)
	}
}

// ---------------------------------------------------------------------------
// taoDetectServices
// ---------------------------------------------------------------------------

func TestTaoDetectServices(t *testing.T) {
	models := []llmprovider.ModelInfo{
		{ID: "deepseek-chat", Capabilities: []string{"chat", "streaming", "vision"}},
		{ID: "whisper-1", Capabilities: []string{"audio"}},
		{ID: "text-embedding-3-large", Capabilities: []string{"embedding"}},
		{ID: "dall-e-3", Capabilities: []string{"image_generation"}},
	}

	svc := taoDetectServices(models)
	if !svc.LLM {
		t.Fatal("expected llm=true")
	}
	if !svc.Search {
		t.Fatal("expected search=true")
	}
	if !svc.OCR {
		t.Fatal("expected ocr=true (vision model)")
	}
	if !svc.Audio {
		t.Fatal("expected audio=true")
	}
	if !svc.Embedding {
		t.Fatal("expected embedding=true")
	}
	if !svc.Image {
		t.Fatal("expected image=true")
	}
}

func TestTaoDetectServices_Empty(t *testing.T) {
	svc := taoDetectServices(nil)
	if svc.LLM {
		t.Fatal("expected llm=false for empty models")
	}
	if !svc.Search {
		t.Fatal("search should default to true")
	}
}

// ---------------------------------------------------------------------------
// taoCallVerify — server error path
// ---------------------------------------------------------------------------

func TestTaoCallVerify_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	result := taoCallVerify(srv.URL, "some-key")
	if result.Valid {
		t.Fatal("expected valid=false on 500")
	}
	if result.Message == "" {
		t.Fatal("expected non-empty error message")
	}
}

func TestTaoCallVerify_ValidFalse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"valid":             false,
			"balance_available": false,
		})
	}))
	defer srv.Close()

	result := taoCallVerify(srv.URL, "some-key")
	if result.Valid {
		t.Fatal("expected valid=false")
	}
	if result.Message != "key is not valid" {
		t.Fatalf("expected 'key is not valid', got %q", result.Message)
	}
}

// ---------------------------------------------------------------------------
// taoFetchModelDefaults — failure paths
// ---------------------------------------------------------------------------

func TestTaoFetchModelDefaults_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	roles := taoFetchModelDefaults(srv.URL)
	if roles != nil {
		t.Fatalf("expected nil on server error, got %v", roles)
	}
}

func TestTaoFetchModelDefaults_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	roles := taoFetchModelDefaults(srv.URL)
	if roles != nil {
		t.Fatalf("expected nil on invalid json, got %v", roles)
	}
}

// ---------------------------------------------------------------------------
// taoServicesFromMap
// ---------------------------------------------------------------------------

func TestTaoServicesFromMap(t *testing.T) {
	m := map[string]interface{}{
		"llm":       true,
		"search":    true,
		"scrape":    false,
		"ocr":       true,
		"image":     false,
		"audio":     true,
		"embedding": false,
	}
	svc := taoServicesFromMap(m)
	if !svc.LLM {
		t.Fatal("expected llm=true")
	}
	if !svc.Search {
		t.Fatal("expected search=true")
	}
	if svc.Scrape {
		t.Fatal("expected scrape=false")
	}
	if !svc.OCR {
		t.Fatal("expected ocr=true")
	}
	if svc.Image {
		t.Fatal("expected image=false")
	}
	if !svc.Audio {
		t.Fatal("expected audio=true")
	}
	if svc.Embedding {
		t.Fatal("expected embedding=false")
	}
}

func TestTaoServicesFromMap_Nil(t *testing.T) {
	svc := taoServicesFromMap(nil)
	if svc.LLM || svc.Search || svc.Scrape || svc.OCR || svc.Image || svc.Audio || svc.Embedding {
		t.Fatal("expected all false for nil map")
	}
}

// ---------------------------------------------------------------------------
// taoFetchBalance — invalid JSON
// ---------------------------------------------------------------------------

func TestTaoFetchBalance_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	bal, avail := taoFetchBalance(srv.URL, "key", 5*time.Second)
	if avail {
		t.Fatal("expected balance_available=false on invalid json")
	}
	if bal != nil {
		t.Fatal("expected nil balance on invalid json")
	}
}

// ---------------------------------------------------------------------------
// taoCallVerify — connection refused
// ---------------------------------------------------------------------------

func TestTaoCallVerify_ConnectionRefused(t *testing.T) {
	result := taoCallVerify("http://127.0.0.1:1", "key")
	if result.Valid {
		t.Fatal("expected valid=false on connection refused")
	}
	if result.Message == "" {
		t.Fatal("expected non-empty error message")
	}
}

// ---------------------------------------------------------------------------
// taoCallVerify — 401 with empty message (fallback path)
// ---------------------------------------------------------------------------

func TestTaoCallVerify_UnauthorizedNoMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{}}`))
	}))
	defer srv.Close()

	result := taoCallVerify(srv.URL, "key")
	if result.Valid {
		t.Fatal("expected valid=false")
	}
	if result.Message == "" {
		t.Fatal("expected non-empty fallback message")
	}
	// Fallback message should contain HTTP status
	if result.Message != "invalid API key (HTTP 401)" {
		t.Fatalf("expected fallback message, got %q", result.Message)
	}
}

func TestTaoCallVerify_ForbiddenStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	result := taoCallVerify(srv.URL, "key")
	if result.Valid {
		t.Fatal("expected valid=false on 403")
	}
}

func TestTaoCallVerify_InvalidResponseJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	result := taoCallVerify(srv.URL, "key")
	if result.Valid {
		t.Fatal("expected valid=false on invalid json")
	}
	if result.Message != "invalid response" {
		t.Fatalf("expected 'invalid response', got %q", result.Message)
	}
}

// ---------------------------------------------------------------------------
// scopeFromAuth
// ---------------------------------------------------------------------------

func TestScopeFromAuth_User(t *testing.T) {
	info := &oauthTypes.AuthorizedInfo{UserID: "u1", TeamID: ""}
	s := scopeFromAuth(info)
	if s.Scope != setting.ScopeUser {
		t.Fatalf("expected ScopeUser, got %v", s.Scope)
	}
	if s.UserID != "u1" {
		t.Fatalf("expected UserID=u1, got %s", s.UserID)
	}
}

func TestScopeFromAuth_Team(t *testing.T) {
	info := &oauthTypes.AuthorizedInfo{UserID: "u1", TeamID: "t1"}
	s := scopeFromAuth(info)
	if s.Scope != setting.ScopeTeam {
		t.Fatalf("expected ScopeTeam, got %v", s.Scope)
	}
	if s.TeamID != "t1" {
		t.Fatalf("expected TeamID=t1, got %s", s.TeamID)
	}
}

// ---------------------------------------------------------------------------
// taoFetchBalance — connection refused
// ---------------------------------------------------------------------------

func TestTaoFetchBalance_ConnectionRefused(t *testing.T) {
	bal, avail := taoFetchBalance("http://127.0.0.1:1", "key", 2*time.Second)
	if avail {
		t.Fatal("expected balance_available=false")
	}
	if bal != nil {
		t.Fatal("expected nil balance")
	}
}

// ---------------------------------------------------------------------------
// taoFetchModelDefaults — connection refused
// ---------------------------------------------------------------------------

func TestTaoFetchModelDefaults_ConnectionRefused(t *testing.T) {
	roles := taoFetchModelDefaults("http://127.0.0.1:1")
	if roles != nil {
		t.Fatalf("expected nil on connection refused, got %v", roles)
	}
}
