package setting

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/store"
	registry "github.com/yaoapp/yao/setting"
	"github.com/yaoapp/yao/tools/websearch"
)

func TestParallelSettingsSelection(t *testing.T) {
	t.Setenv("TAVILY_API_KEY", "")
	oldGlobal := registry.Global
	oldStore := store.Pools["__yao.store"]
	oldCache := store.Pools["__yao.cache"]
	t.Cleanup(func() {
		registry.Global = oldGlobal
		store.Pools["__yao.store"] = oldStore
		store.Pools["__yao.cache"] = oldCache
	})
	s, err := store.New(nil, store.Option{"size": 100})
	if err != nil {
		t.Fatal(err)
	}
	store.Pools["__yao.store"] = s
	delete(store.Pools, "__yao.cache")
	if err := registry.Init(); err != nil {
		t.Fatal(err)
	}
	call := func(handler gin.HandlerFunc, body string, key string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPut, "/setting/search", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set("__user_id", "parallel-test")
		c.Params = gin.Params{{Key: "key", Value: key}}
		handler(c)
		return w
	}
	preset := searchFindPreset("parallel")
	if preset == nil || len(preset.Tools) != 1 || preset.Tools[0] != "web_search" {
		t.Fatal("Parallel not in native search presets")
	}
	if len(preset.Fields) != 1 || preset.Fields[0].Key != "api_key" || !preset.Fields[0].Optional {
		t.Fatal("Parallel's anonymous preset must advertise an optional API key")
	}
	if w := call(handleSearchProviderToggle, `{"enabled":true}`, "parallel"); w.Code != http.StatusOK {
		t.Fatalf("enable: %d %s", w.Code, w.Body.String())
	}
	if w := call(handleSearchToolAssignment, `{"web_search":"parallel"}`, ""); w.Code != http.StatusOK {
		t.Fatalf("assign: %d %s", w.Code, w.Body.String())
	}
	// A blank query fails before any network call, proving the settings API reaches
	// the Parallel adapter through the same canonical entrypoint as agent searches.
	results := websearch.Search(" ", 1, "parallel-test", "")
	if len(results) != 1 || !strings.Contains(results[0].Content, "parallel search query is empty") {
		t.Fatalf("settings did not select Parallel: %+v", results)
	}
	if w := call(handleSearchProviderToggle, `{"enabled":false}`, "parallel"); w.Code != http.StatusOK {
		t.Fatal("disable failed")
	}
	if r := websearch.Search(" ", 1, "parallel-test", ""); len(r) != 0 {
		t.Fatalf("disable did not restore keyless Tavily baseline: %+v", r)
	}
}
