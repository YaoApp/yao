//go:build integration

package agent_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yaoapp/yao/agent"
	"github.com/yaoapp/yao/setting"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestSyncSearchDefaults(t *testing.T) {
	testprepare.PrepareSandbox(t)
	appDir := testprepare.AgentAppDir()
	agentDir := filepath.Join(appDir, "agent")

	t.Run("websearch_and_webfetch", func(t *testing.T) {
		wsFile := filepath.Join(agentDir, "websearch.yml")
		wfFile := filepath.Join(agentDir, "webfetch.yml")
		t.Cleanup(func() {
			os.Remove(wsFile)
			os.Remove(wfFile)
		})

		os.Setenv("TEST_SERPER_KEY", "sk-test-serper-123")
		os.Setenv("TEST_BD_KEY", "sk-test-bd-456")
		t.Cleanup(func() {
			os.Unsetenv("TEST_SERPER_KEY")
			os.Unsetenv("TEST_BD_KEY")
		})

		wsContent := []byte("default: serper\nproviders:\n  serper:\n    api_key: $ENV.TEST_SERPER_KEY\n")
		if err := os.WriteFile(wsFile, wsContent, 0644); err != nil {
			t.Fatalf("write websearch.yml: %v", err)
		}

		wfContent := []byte("default: brightdata\nproviders:\n  brightdata:\n    api_key: $ENV.TEST_BD_KEY\n    zone: web_unlocker1\n")
		if err := os.WriteFile(wfFile, wfContent, 0644); err != nil {
			t.Fatalf("write webfetch.yml: %v", err)
		}

		if err := agent.SyncSearchDefaults(); err != nil {
			t.Fatalf("SyncSearchDefaults: %v", err)
		}

		systemScope := setting.ScopeID{Scope: setting.ScopeSystem}

		assignment, err := setting.Global.Get(systemScope, "search.tool_assignment")
		if err != nil {
			t.Fatalf("get search.tool_assignment: %v", err)
		}
		if v, ok := assignment["web_search"].(string); !ok || v != "serper" {
			t.Errorf("web_search = %v, want 'serper'", assignment["web_search"])
		}
		if v, ok := assignment["web_scrape"].(string); !ok || v != "brightdata" {
			t.Errorf("web_scrape = %v, want 'brightdata'", assignment["web_scrape"])
		}

		serperData, err := setting.Global.Get(systemScope, "search.providers.serper")
		if err != nil {
			t.Fatalf("get search.providers.serper: %v", err)
		}
		fieldValues, ok := serperData["field_values"].(map[string]interface{})
		if !ok {
			t.Fatalf("serper field_values not a map: %T", serperData["field_values"])
		}
		apiKey, _ := fieldValues["api_key"].(string)
		if apiKey == "" {
			t.Fatal("serper api_key is empty")
		}
		if apiKey == "sk-test-serper-123" {
			t.Error("serper api_key should be encrypted, got plaintext")
		}

		bdData, err := setting.Global.Get(systemScope, "search.providers.brightdata")
		if err != nil {
			t.Fatalf("get search.providers.brightdata: %v", err)
		}
		bdFields, ok := bdData["field_values"].(map[string]interface{})
		if !ok {
			t.Fatalf("brightdata field_values not a map: %T", bdData["field_values"])
		}
		if zone, _ := bdFields["zone"].(string); zone != "web_unlocker1" {
			t.Errorf("brightdata zone = %q, want 'web_unlocker1'", zone)
		}
	})

	t.Run("empty_api_key_skipped", func(t *testing.T) {
		wsFile := filepath.Join(agentDir, "websearch.yml")
		t.Cleanup(func() { os.Remove(wsFile) })

		os.Unsetenv("TEST_EMPTY_KEY")

		wsContent := []byte("default: tavily\nproviders:\n  tavily:\n    api_key: $ENV.TEST_EMPTY_KEY\n")
		if err := os.WriteFile(wsFile, wsContent, 0644); err != nil {
			t.Fatalf("write websearch.yml: %v", err)
		}

		// Delete any existing tavily provider data
		systemScope := setting.ScopeID{Scope: setting.ScopeSystem}
		_ = setting.Global.Delete(systemScope, "search.providers.tavily")

		if err := agent.SyncSearchDefaults(); err != nil {
			t.Fatalf("SyncSearchDefaults: %v", err)
		}

		_, err := setting.Global.Get(systemScope, "search.providers.tavily")
		if err == nil {
			t.Error("tavily provider should not be written when api_key is empty")
		}
	})

	t.Run("direct_default_no_providers", func(t *testing.T) {
		wfFile := filepath.Join(agentDir, "webfetch.yml")
		t.Cleanup(func() { os.Remove(wfFile) })

		wfContent := []byte("default: direct\n")
		if err := os.WriteFile(wfFile, wfContent, 0644); err != nil {
			t.Fatalf("write webfetch.yml: %v", err)
		}

		if err := agent.SyncSearchDefaults(); err != nil {
			t.Fatalf("SyncSearchDefaults: %v", err)
		}

		systemScope := setting.ScopeID{Scope: setting.ScopeSystem}
		assignment, err := setting.Global.Get(systemScope, "search.tool_assignment")
		if err != nil {
			t.Fatalf("get search.tool_assignment: %v", err)
		}
		if v, ok := assignment["web_scrape"].(string); !ok || v != "direct" {
			t.Errorf("web_scrape = %v, want 'direct'", assignment["web_scrape"])
		}
	})

	t.Run("partial_fields_written", func(t *testing.T) {
		// After skip-condition refactor: providers with at least one non-empty
		// field value are written even if api_key is absent (partial config).
		wfFile := filepath.Join(agentDir, "webfetch.yml")
		t.Cleanup(func() { os.Remove(wfFile) })

		wfContent := []byte("default: brightdata\nproviders:\n  brightdata:\n    zone: web_unlocker1\n")
		if err := os.WriteFile(wfFile, wfContent, 0644); err != nil {
			t.Fatalf("write webfetch.yml: %v", err)
		}

		systemScope := setting.ScopeID{Scope: setting.ScopeSystem}
		_ = setting.Global.Delete(systemScope, "search.providers.brightdata")

		if err := agent.SyncSearchDefaults(); err != nil {
			t.Fatalf("SyncSearchDefaults: %v", err)
		}

		saved, err := setting.Global.Get(systemScope, "search.providers.brightdata")
		if err != nil {
			t.Fatalf("brightdata provider should be written when zone has a value: %v", err)
		}
		fv, ok := saved["field_values"].(map[string]interface{})
		if !ok {
			t.Fatalf("field_values not a map: %T", saved["field_values"])
		}
		if zone, _ := fv["zone"].(string); zone != "web_unlocker1" {
			t.Errorf("zone = %q, want 'web_unlocker1'", zone)
		}
	})

	t.Run("all_fields_empty_skipped", func(t *testing.T) {
		// Provider with all fields empty (after ENV resolution) is skipped.
		wfFile := filepath.Join(agentDir, "webfetch.yml")
		t.Cleanup(func() { os.Remove(wfFile) })

		os.Unsetenv("TEST_EMPTY_BD_KEY")
		os.Unsetenv("TEST_EMPTY_BD_ZONE")

		wfContent := []byte("default: brightdata\nproviders:\n  brightdata:\n    api_key: $ENV.TEST_EMPTY_BD_KEY\n    zone: $ENV.TEST_EMPTY_BD_ZONE\n")
		if err := os.WriteFile(wfFile, wfContent, 0644); err != nil {
			t.Fatalf("write webfetch.yml: %v", err)
		}

		systemScope := setting.ScopeID{Scope: setting.ScopeSystem}
		_ = setting.Global.Delete(systemScope, "search.providers.brightdata")

		if err := agent.SyncSearchDefaults(); err != nil {
			t.Fatalf("SyncSearchDefaults: %v", err)
		}

		_, err := setting.Global.Get(systemScope, "search.providers.brightdata")
		if err == nil {
			t.Error("brightdata provider should not be written when all fields are empty")
		}
	})

	t.Run("unknown_default_ignored", func(t *testing.T) {
		wsFile := filepath.Join(agentDir, "websearch.yml")
		t.Cleanup(func() { os.Remove(wsFile) })

		wsContent := []byte("default: unknown_provider\n")
		if err := os.WriteFile(wsFile, wsContent, 0644); err != nil {
			t.Fatalf("write websearch.yml: %v", err)
		}

		// Set a known value first
		systemScope := setting.ScopeID{Scope: setting.ScopeSystem}
		setting.Global.Set(systemScope, "search.tool_assignment", map[string]interface{}{
			"web_search": "serper",
		})

		if err := agent.SyncSearchDefaults(); err != nil {
			t.Fatalf("SyncSearchDefaults: %v", err)
		}

		assignment, err := setting.Global.Get(systemScope, "search.tool_assignment")
		if err != nil {
			t.Fatalf("get search.tool_assignment: %v", err)
		}
		if v, _ := assignment["web_search"].(string); v != "serper" {
			t.Errorf("web_search = %v, want 'serper' (should not be overwritten by unknown default)", v)
		}
	})
}

func TestSyncOCRDefaults(t *testing.T) {
	testprepare.PrepareSandbox(t)
	appDir := testprepare.AgentAppDir()
	agentDir := filepath.Join(appDir, "agent")

	t.Run("baidu_provider_with_env", func(t *testing.T) {
		ocrFile := filepath.Join(agentDir, "ocr.yml")
		t.Cleanup(func() { os.Remove(ocrFile) })

		os.Setenv("TEST_BAIDU_OCR_KEY", "sk-test-baidu-ocr")
		os.Setenv("TEST_BAIDU_OCR_SECRET", "sk-test-baidu-secret")
		t.Cleanup(func() {
			os.Unsetenv("TEST_BAIDU_OCR_KEY")
			os.Unsetenv("TEST_BAIDU_OCR_SECRET")
		})

		ocrContent := []byte("default: baidu\nproviders:\n  baidu:\n    api_key: $ENV.TEST_BAIDU_OCR_KEY\n    secret_key: $ENV.TEST_BAIDU_OCR_SECRET\n")
		if err := os.WriteFile(ocrFile, ocrContent, 0644); err != nil {
			t.Fatalf("write ocr.yml: %v", err)
		}

		if err := agent.SyncOCRDefaults(); err != nil {
			t.Fatalf("SyncOCRDefaults: %v", err)
		}

		systemScope := setting.ScopeID{Scope: setting.ScopeSystem}

		// Verify tool assignment
		assignment, err := setting.Global.Get(systemScope, "ocr.tool_assignment")
		if err != nil {
			t.Fatalf("get ocr.tool_assignment: %v", err)
		}
		if v, ok := assignment["ocr_recognize"].(string); !ok || v != "baidu" {
			t.Errorf("ocr_recognize = %v, want 'baidu'", assignment["ocr_recognize"])
		}

		// Verify provider data
		baiduData, err := setting.Global.Get(systemScope, "ocr.providers.baidu")
		if err != nil {
			t.Fatalf("get ocr.providers.baidu: %v", err)
		}
		fv, ok := baiduData["field_values"].(map[string]interface{})
		if !ok {
			t.Fatalf("baidu field_values not a map: %T", baiduData["field_values"])
		}

		// api_key and secret_key should be encrypted
		apiKey, _ := fv["api_key"].(string)
		if apiKey == "" {
			t.Fatal("baidu api_key is empty")
		}
		if apiKey == "sk-test-baidu-ocr" {
			t.Error("baidu api_key should be encrypted, got plaintext")
		}
		secretKey, _ := fv["secret_key"].(string)
		if secretKey == "" {
			t.Fatal("baidu secret_key is empty")
		}
		if secretKey == "sk-test-baidu-secret" {
			t.Error("baidu secret_key should be encrypted, got plaintext")
		}

		// enabled and status
		if v, _ := baiduData["enabled"].(bool); !v {
			t.Error("baidu should be enabled")
		}
		if v, _ := baiduData["status"].(string); v != "connected" {
			t.Errorf("baidu status = %q, want 'connected'", v)
		}
	})

	t.Run("paddleocr_not_skipped", func(t *testing.T) {
		// PaddleOCR has only base_url (no api_key). Must not be skipped.
		ocrFile := filepath.Join(agentDir, "ocr.yml")
		t.Cleanup(func() { os.Remove(ocrFile) })

		ocrContent := []byte("default: paddleocr\nproviders:\n  paddleocr:\n    base_url: http://localhost:8080\n")
		if err := os.WriteFile(ocrFile, ocrContent, 0644); err != nil {
			t.Fatalf("write ocr.yml: %v", err)
		}

		systemScope := setting.ScopeID{Scope: setting.ScopeSystem}
		_ = setting.Global.Delete(systemScope, "ocr.providers.paddleocr")

		if err := agent.SyncOCRDefaults(); err != nil {
			t.Fatalf("SyncOCRDefaults: %v", err)
		}

		saved, err := setting.Global.Get(systemScope, "ocr.providers.paddleocr")
		if err != nil {
			t.Fatalf("paddleocr provider should be written (has base_url): %v", err)
		}
		fv, ok := saved["field_values"].(map[string]interface{})
		if !ok {
			t.Fatalf("field_values not a map: %T", saved["field_values"])
		}
		if url, _ := fv["base_url"].(string); url != "http://localhost:8080" {
			t.Errorf("base_url = %q, want 'http://localhost:8080'", url)
		}
		// base_url is not a password field — should be plaintext
		if url, _ := fv["base_url"].(string); url != "http://localhost:8080" {
			t.Errorf("base_url should not be encrypted: %q", url)
		}
	})

	t.Run("empty_env_skipped", func(t *testing.T) {
		ocrFile := filepath.Join(agentDir, "ocr.yml")
		t.Cleanup(func() { os.Remove(ocrFile) })

		os.Unsetenv("TEST_EMPTY_OCR_KEY")
		os.Unsetenv("TEST_EMPTY_OCR_SECRET")

		ocrContent := []byte("default: baidu\nproviders:\n  baidu:\n    api_key: $ENV.TEST_EMPTY_OCR_KEY\n    secret_key: $ENV.TEST_EMPTY_OCR_SECRET\n")
		if err := os.WriteFile(ocrFile, ocrContent, 0644); err != nil {
			t.Fatalf("write ocr.yml: %v", err)
		}

		systemScope := setting.ScopeID{Scope: setting.ScopeSystem}
		_ = setting.Global.Delete(systemScope, "ocr.providers.baidu")

		if err := agent.SyncOCRDefaults(); err != nil {
			t.Fatalf("SyncOCRDefaults: %v", err)
		}

		_, err := setting.Global.Get(systemScope, "ocr.providers.baidu")
		if err == nil {
			t.Error("baidu provider should not be written when all fields are empty")
		}
	})

	t.Run("unknown_provider_skipped", func(t *testing.T) {
		ocrFile := filepath.Join(agentDir, "ocr.yml")
		t.Cleanup(func() { os.Remove(ocrFile) })

		ocrContent := []byte("providers:\n  unknown_ocr:\n    api_key: some-key\n")
		if err := os.WriteFile(ocrFile, ocrContent, 0644); err != nil {
			t.Fatalf("write ocr.yml: %v", err)
		}

		systemScope := setting.ScopeID{Scope: setting.ScopeSystem}
		_ = setting.Global.Delete(systemScope, "ocr.providers.unknown_ocr")

		if err := agent.SyncOCRDefaults(); err != nil {
			t.Fatalf("SyncOCRDefaults: %v", err)
		}

		_, err := setting.Global.Get(systemScope, "ocr.providers.unknown_ocr")
		if err == nil {
			t.Error("unknown provider should not be written")
		}
	})

	t.Run("unknown_default_ignored", func(t *testing.T) {
		ocrFile := filepath.Join(agentDir, "ocr.yml")
		t.Cleanup(func() { os.Remove(ocrFile) })

		ocrContent := []byte("default: unknown_provider\n")
		if err := os.WriteFile(ocrFile, ocrContent, 0644); err != nil {
			t.Fatalf("write ocr.yml: %v", err)
		}

		// Set a known value first
		systemScope := setting.ScopeID{Scope: setting.ScopeSystem}
		setting.Global.Set(systemScope, "ocr.tool_assignment", map[string]interface{}{
			"ocr_recognize": "baidu",
		})

		if err := agent.SyncOCRDefaults(); err != nil {
			t.Fatalf("SyncOCRDefaults: %v", err)
		}

		assignment, err := setting.Global.Get(systemScope, "ocr.tool_assignment")
		if err != nil {
			t.Fatalf("get ocr.tool_assignment: %v", err)
		}
		if v, _ := assignment["ocr_recognize"].(string); v != "baidu" {
			t.Errorf("ocr_recognize = %v, want 'baidu' (should not be overwritten by unknown default)", v)
		}
	})
}
