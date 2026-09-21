package websearch

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yaoapp/gou/mcp/types"
	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/yao/setting"
	"github.com/yaoapp/yao/share"
)

func TestParallelSearchMCP(t *testing.T) {
	for _, key := range []string{"", "configured-key"} {
		t.Run(key, func(t *testing.T) {
			called := false
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}
				if r.Header.Get("User-Agent") != "Yao/"+share.VERSION {
					t.Error("project User-Agent missing")
				}
				if r.Header.Get("x-api-key") != key || r.Header.Get("Authorization") != "" {
					t.Error("unexpected authentication")
				}
				var req struct {
					ID     json.RawMessage        `json:"id"`
					Method string                 `json:"method"`
					Params map[string]interface{} `json:"params"`
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Error(err)
					return
				}
				if len(req.ID) == 0 {
					w.WriteHeader(http.StatusAccepted)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				var result interface{}
				switch req.Method {
				case "initialize":
					result = map[string]interface{}{"protocolVersion": "2025-03-26", "capabilities": map[string]interface{}{"tools": map[string]interface{}{}}, "serverInfo": map[string]string{"name": "fixture", "version": "1"}}
				case "tools/list":
					result = map[string]interface{}{"tools": []interface{}{map[string]interface{}{"name": "web_search", "inputSchema": map[string]string{"type": "object"}}}}
				case "tools/call":
					called = true
					if req.Params["name"] != "web_search" {
						t.Error("wrong tool")
					}
					args := req.Params["arguments"].(map[string]interface{})
					if args["objective"] != "go generics" || len(args["search_queries"].([]interface{})) != 1 {
						t.Error("wrong search arguments")
					}
					if _, ok := args["query"]; ok {
						t.Error("generic MCP argument leaked")
					}
					result = map[string]interface{}{"content": []interface{}{map[string]string{"type": "text", "text": `{"results":[{"title":"Go","url":"https://go.dev","excerpts":["First","Second"]},{"url":"https://go.dev/blog","excerpts":[]}]}`}}}
				default:
					t.Errorf("unexpected method %s", req.Method)
				}
				json.NewEncoder(w).Encode(map[string]interface{}{"jsonrpc": "2.0", "id": req.ID, "result": result})
			}))
			defer srv.Close()
			results, err := parallelSearchAt(srv.URL, "go generics", 1, key)
			if err != nil {
				t.Fatal(err)
			}
			if !called || len(results) != 1 || results[0].URL != "https://go.dev" || results[0].Content != "First\n\nSecond" {
				t.Fatalf("unexpected results %+v", results)
			}
		})
	}
}

func TestParallelResultErrors(t *testing.T) {
	for _, tc := range []struct {
		name, text         string
		isError, wantError bool
	}{
		{"empty", `{"results":[]}`, false, false},
		{"tool error", `{"results":[]}`, true, true},
		{"malformed", `not JSON`, false, true},
		{"missing results", `{}`, false, true},
		{"missing citation", `{"results":[{"excerpts":["uncited"]}]}`, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseParallelResult(&types.CallToolResponse{IsError: tc.isError, Content: []types.ToolContent{{Type: types.ToolContentTypeText, Text: tc.text}}}, 5)
			if (err != nil) != tc.wantError {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestParallelSelection(t *testing.T) {
	oldGlobal := setting.Global
	oldStore := store.Pools["__yao.store"]
	oldCache := store.Pools["__yao.cache"]
	t.Cleanup(func() {
		setting.Global = oldGlobal
		store.Pools["__yao.store"] = oldStore
		store.Pools["__yao.cache"] = oldCache
	})
	s, err := store.New(nil, store.Option{"size": 100})
	if err != nil {
		t.Fatal(err)
	}
	store.Pools["__yao.store"] = s
	delete(store.Pools, "__yao.cache")
	if err := setting.Init(); err != nil {
		t.Fatal(err)
	}
	if cfg := getConfig("", ""); cfg.Provider != "tavily" {
		t.Fatal("default changed")
	}
	scope := setting.ScopeID{Scope: setting.ScopeSystem}
	if _, err := setting.Global.Set(scope, "search.tool_assignment", map[string]interface{}{"web_search": "parallel"}); err != nil {
		t.Fatal(err)
	}
	cfg := getConfig("", "")
	if cfg.Provider != "parallel" || cfg.APIKey != "" {
		t.Fatalf("anonymous selection: %+v", cfg)
	}
	if _, err := setting.Global.Set(setting.ScopeID{Scope: setting.ScopeUser, UserID: "test-user"}, "search.tool_assignment", map[string]interface{}{"web_search": "serper"}); err != nil {
		t.Fatal(err)
	}
	if cfg := getConfig("test-user", ""); cfg.Provider != "serper" {
		t.Fatal("user override lost")
	}
	results := Search(" ", 1, "", "")
	if len(results) != 1 || results[0].Title != "Error" || !strings.Contains(results[0].Content, "empty") {
		t.Fatalf("canonical selection did not reach Parallel: %+v", results)
	}
}
