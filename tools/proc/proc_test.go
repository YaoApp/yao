package proc

import (
	"sync"
	"testing"

	"github.com/yaoapp/gou/process"
)

func TestDefaultAllowed(t *testing.T) {
	resetConfig()
	allowed := []string{
		"http.Get",
		"http.post",
		"encoding.json.Encode",
		"encoding.base64.Decode",
		"json.parse",
		"json.validate",
		"text.extract",
		"text.htmltomarkdown",
	}
	for _, name := range allowed {
		if !isAllowedProcess(name) {
			t.Errorf("expected %q to be allowed by default", name)
		}
	}
}

func TestDefaultBlocked(t *testing.T) {
	resetConfig()
	blocked := []string{
		"utils.str.Join",
		"utils.app.Inspect",
		"models.user.Find",
		"model.load",
		"schemas.default.tablecreate",
		"stores.cache.Set",
		"flows.login.Run",
		"scripts.helper.Format",
		"yao.sys.Exec",
		"yao.env.Get",
		"tools.web_search",
		"fs.system.readfile",
		"unknown.process",
	}
	for _, name := range blocked {
		if isAllowedProcess(name) {
			t.Errorf("expected %q to be blocked by default", name)
		}
	}
}

func TestAppConfigReplacesDefault(t *testing.T) {
	resetConfig()
	config = &Config{
		ProcessCall: ProcessCallConfig{
			Allowed: []string{
				"models.*",
				"scripts.*",
				"flows.*",
				"http.*",
				"stores.cache.*",
			},
		},
	}

	allowed := []string{
		"models.user.Find",
		"models.order.Create",
		"scripts.helper.Format",
		"flows.login.Run",
		"http.Get",
		"stores.cache.Set",
		"stores.cache.Get",
	}
	for _, name := range allowed {
		if !isAllowedProcess(name) {
			t.Errorf("expected %q to be allowed with app config", name)
		}
	}

	blocked := []string{
		"encoding.json.Encode",
		"json.parse",
		"text.extract",
		"utils.str.Join",
		"stores.session.Set",
		"yao.sys.Exec",
	}
	for _, name := range blocked {
		if isAllowedProcess(name) {
			t.Errorf("expected %q to be blocked with app config (not in tools.yml)", name)
		}
	}
}

func TestExactMatch(t *testing.T) {
	resetConfig()
	config = &Config{
		ProcessCall: ProcessCallConfig{
			Allowed: []string{
				"models.user.Find",
				"models.user.Get",
				"scripts.auth.Login",
			},
		},
	}

	allowed := []string{
		"models.user.Find",
		"models.user.Get",
		"scripts.auth.Login",
	}
	for _, name := range allowed {
		if !isAllowedProcess(name) {
			t.Errorf("expected %q to be allowed by exact match", name)
		}
	}

	blocked := []string{
		"models.user.Create",
		"models.order.Find",
		"scripts.auth.Logout",
		"scripts.helper.Run",
	}
	for _, name := range blocked {
		if isAllowedProcess(name) {
			t.Errorf("expected %q to be blocked (not in exact match list)", name)
		}
	}
}

func TestCaseInsensitive(t *testing.T) {
	resetConfig()

	if !isAllowedProcess("HTTP.GET") {
		t.Error("expected HTTP.GET to be allowed (case insensitive)")
	}
	if !isAllowedProcess("Json.Parse") {
		t.Error("expected Json.Parse to be allowed (case insensitive)")
	}

	resetConfig()
	config = &Config{
		ProcessCall: ProcessCallConfig{
			Allowed: []string{"Models.*", "scripts.Auth.Login"},
		},
	}
	if !isAllowedProcess("models.user.Find") {
		t.Error("expected models.user.Find to match Models.* (case insensitive)")
	}
	if !isAllowedProcess("Scripts.Auth.Login") {
		t.Error("expected Scripts.Auth.Login to match scripts.Auth.Login (case insensitive)")
	}
}

func TestMixedRules(t *testing.T) {
	resetConfig()
	config = &Config{
		ProcessCall: ProcessCallConfig{
			Allowed: []string{
				"models.*",
				"scripts.auth.Login",
				"http.*",
			},
		},
	}

	allowed := []string{
		"models.user.Find",
		"models.order.Create",
		"scripts.auth.Login",
		"http.Get",
	}
	for _, name := range allowed {
		if !isAllowedProcess(name) {
			t.Errorf("expected %q to be allowed with mixed rules", name)
		}
	}

	blocked := []string{
		"scripts.auth.Logout",
		"scripts.helper.Run",
		"flows.login.Run",
	}
	for _, name := range blocked {
		if isAllowedProcess(name) {
			t.Errorf("expected %q to be blocked with mixed rules", name)
		}
	}
}

func TestEmptyConfig(t *testing.T) {
	resetConfig()
	config = &Config{
		ProcessCall: ProcessCallConfig{
			Allowed: []string{},
		},
	}

	// Empty allowed list in config means nothing is allowed — falls through to default
	if !isAllowedProcess("http.Get") {
		t.Error("expected http.Get to be allowed when config has empty allowed list (default fallback)")
	}
}

func TestAllowedHandlerListDefault(t *testing.T) {
	resetConfig()
	p := &process.Process{Args: []interface{}{}}
	result := AllowedHandler(p)
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	rules, ok := m["rules"]
	if !ok {
		t.Fatal("expected 'rules' key in result")
	}
	ruleSlice, ok := rules.([]string)
	if !ok {
		t.Fatalf("expected []string for rules, got %T", rules)
	}
	if len(ruleSlice) != len(defaultAllowed) {
		t.Errorf("expected %d default rules, got %d", len(defaultAllowed), len(ruleSlice))
	}
	for i, r := range defaultAllowed {
		if ruleSlice[i] != r {
			t.Errorf("rule[%d]: expected %q, got %q", i, r, ruleSlice[i])
		}
	}
}

func TestAllowedHandlerListCustomConfig(t *testing.T) {
	resetConfig()
	config = &Config{
		ProcessCall: ProcessCallConfig{
			Allowed: []string{"models.*", "scripts.*", "http.*"},
		},
	}
	p := &process.Process{Args: []interface{}{}}
	result := AllowedHandler(p)
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	rules := m["rules"].([]string)
	expected := []string{"models.*", "scripts.*", "http.*"}
	if len(rules) != len(expected) {
		t.Errorf("expected %d rules, got %d", len(expected), len(rules))
	}
	for i, r := range expected {
		if rules[i] != r {
			t.Errorf("rule[%d]: expected %q, got %q", i, r, rules[i])
		}
	}
}

func TestAllowedHandlerCheckAllowed(t *testing.T) {
	resetConfig()
	p := &process.Process{Args: []interface{}{"http.Get"}}
	result := AllowedHandler(p)
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	if m["name"] != "http.Get" {
		t.Errorf("expected name 'http.Get', got %v", m["name"])
	}
	if m["allowed"] != true {
		t.Error("expected http.Get to be allowed")
	}
}

func TestAllowedHandlerCheckBlocked(t *testing.T) {
	resetConfig()
	p := &process.Process{Args: []interface{}{"models.user.Find"}}
	result := AllowedHandler(p)
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	if m["name"] != "models.user.Find" {
		t.Errorf("expected name 'models.user.Find', got %v", m["name"])
	}
	if m["allowed"] != false {
		t.Error("expected models.user.Find to be blocked by default")
	}
}

func TestAllowedHandlerCheckWithConfig(t *testing.T) {
	resetConfig()
	config = &Config{
		ProcessCall: ProcessCallConfig{
			Allowed: []string{"models.*", "scripts.auth.Login"},
		},
	}

	// Prefix match
	p := &process.Process{Args: []interface{}{"models.user.Find"}}
	result := AllowedHandler(p).(map[string]interface{})
	if result["allowed"] != true {
		t.Error("expected models.user.Find to be allowed with config")
	}

	// Exact match
	p = &process.Process{Args: []interface{}{"scripts.auth.Login"}}
	result = AllowedHandler(p).(map[string]interface{})
	if result["allowed"] != true {
		t.Error("expected scripts.auth.Login to be allowed with config")
	}

	// Not in config
	p = &process.Process{Args: []interface{}{"http.Get"}}
	result = AllowedHandler(p).(map[string]interface{})
	if result["allowed"] != false {
		t.Error("expected http.Get to be blocked (not in custom config)")
	}
}

func resetConfig() {
	config = nil
	configOnce = sync.Once{}
}
