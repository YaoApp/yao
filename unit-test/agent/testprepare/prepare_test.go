package testprepare

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	code := m.Run()
	Cleanup()
	os.Exit(code)
}

func TestLoadAgentTestEnv_GeneratesAppEnv(t *testing.T) {
	// Clean up any previous app/.env
	envPath := filepath.Join(agentAppDir, ".env")
	os.Remove(envPath)

	err := loadAgentTestEnv()
	if err != nil {
		t.Fatalf("loadAgentTestEnv: %v", err)
	}

	data, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("read app/.env: %v", err)
	}
	content := string(data)

	requiredKeys := []string{
		"YAO_DB_DRIVER=",
		"MOCK_LLM_HOST=",
		"OPENAI_API_KEY=",
		"ANTHROPIC_API_KEY=",
		"DEEPSEEK_V4_API_KEY=",
		"SERPAPI_API_KEY=",
	}
	for _, key := range requiredKeys {
		if !strings.Contains(content, key) {
			t.Errorf("app/.env missing key prefix: %s", key)
		}
	}

	// TEST_* and SANDBOX_TEST_* should NOT appear in app/.env
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		k := strings.SplitN(line, "=", 2)[0]
		if strings.HasPrefix(k, "TEST_") || strings.HasPrefix(k, "SANDBOX_TEST_") {
			t.Errorf("app/.env should not contain orchestration key: %s", k)
		}
	}
}

func TestLoadAgentTestEnv_EnvFileWins(t *testing.T) {
	// Simulate external env pollution
	os.Setenv("YAO_DB_DRIVER", "mysql")
	os.Setenv("YAO_PORT", "9999")
	defer os.Unsetenv("YAO_DB_DRIVER")
	defer os.Unsetenv("YAO_PORT")

	err := loadAgentTestEnv()
	if err != nil {
		t.Fatalf("loadAgentTestEnv: %v", err)
	}

	// agent-test.env says sqlite3 — it should override the external "mysql"
	if v := os.Getenv("YAO_DB_DRIVER"); v == "mysql" {
		t.Error("env file did not override YAO_DB_DRIVER: external pollution leaked")
	}

	// Verify app/.env has the correct value from agent-test.env
	data, err := os.ReadFile(filepath.Join(agentAppDir, ".env"))
	if err != nil {
		t.Fatalf("read app/.env: %v", err)
	}
	if strings.Contains(string(data), "YAO_DB_DRIVER=mysql") {
		t.Error("app/.env contains polluted value YAO_DB_DRIVER=mysql")
	}
}

func TestLoadAgentTestEnv_SetsPathVars(t *testing.T) {
	err := loadAgentTestEnv()
	if err != nil {
		t.Fatalf("loadAgentTestEnv: %v", err)
	}

	for _, key := range []string{"YAO_ROOT", "YAO_AGENT_TEST_APPLICATION", "YAO_TEST_APPLICATION"} {
		v := os.Getenv(key)
		if v == "" {
			t.Errorf("%s not set", key)
			continue
		}
		if v != agentAppDir {
			t.Errorf("%s = %q, want %q", key, v, agentAppDir)
		}
	}
}

func TestParseEnvFile(t *testing.T) {
	envFile := filepath.Join(yaoSrcRoot, "unit-test", "agent", "env", "agent-test.env")
	pairs, err := parseEnvFile(envFile)
	if err != nil {
		t.Fatalf("parseEnvFile: %v", err)
	}

	if len(pairs) == 0 {
		t.Fatal("parseEnvFile returned empty map")
	}

	if v, ok := pairs["YAO_DB_DRIVER"]; !ok {
		t.Error("missing YAO_DB_DRIVER")
	} else if v != "sqlite3" {
		t.Errorf("YAO_DB_DRIVER: got %q, want %q", v, "sqlite3")
	}

	if _, ok := pairs["MOCK_LLM_HOST"]; !ok {
		t.Error("missing MOCK_LLM_HOST")
	}

	if _, ok := pairs["MOCK_LLM_PORT"]; !ok {
		t.Error("missing MOCK_LLM_PORT")
	}
}

func TestPrepareSandbox_ReturnsIdentity(t *testing.T) {
	identity := PrepareSandbox(t)
	if identity == nil {
		t.Fatal("PrepareSandbox returned nil identity — SetupTestUsers likely failed")
	}

	fields := map[string]string{
		"AlphaTeamID":       identity.AlphaTeamID,
		"AlphaOwnerUserID":  identity.AlphaOwnerUserID,
		"AlphaAdminUserID":  identity.AlphaAdminUserID,
		"AlphaMemberUserID": identity.AlphaMemberUserID,

		"BetaOpenAITeamID":       identity.BetaOpenAITeamID,
		"BetaOpenAIOwnerUserID":  identity.BetaOpenAIOwnerUserID,
		"BetaOpenAIAdminUserID":  identity.BetaOpenAIAdminUserID,
		"BetaOpenAIMemberUserID": identity.BetaOpenAIMemberUserID,

		"BetaAnthropicTeamID":       identity.BetaAnthropicTeamID,
		"BetaAnthropicOwnerUserID":  identity.BetaAnthropicOwnerUserID,
		"BetaAnthropicAdminUserID":  identity.BetaAnthropicAdminUserID,
		"BetaAnthropicMemberUserID": identity.BetaAnthropicMemberUserID,

		"BetaHaikuTeamID":       identity.BetaHaikuTeamID,
		"BetaHaikuOwnerUserID":  identity.BetaHaikuOwnerUserID,
		"BetaHaikuAdminUserID":  identity.BetaHaikuAdminUserID,
		"BetaHaikuMemberUserID": identity.BetaHaikuMemberUserID,

		"BetaGPT4oTeamID":       identity.BetaGPT4oTeamID,
		"BetaGPT4oOwnerUserID":  identity.BetaGPT4oOwnerUserID,
		"BetaGPT4oAdminUserID":  identity.BetaGPT4oAdminUserID,
		"BetaGPT4oMemberUserID": identity.BetaGPT4oMemberUserID,
	}
	for name, val := range fields {
		if val == "" {
			t.Errorf("%s is empty", name)
		}
	}

	t.Logf("Alpha:         team=%s owner=%s admin=%s member=%s",
		identity.AlphaTeamID, identity.AlphaOwnerUserID, identity.AlphaAdminUserID, identity.AlphaMemberUserID)
	t.Logf("BetaOpenAI:    team=%s owner=%s admin=%s member=%s",
		identity.BetaOpenAITeamID, identity.BetaOpenAIOwnerUserID, identity.BetaOpenAIAdminUserID, identity.BetaOpenAIMemberUserID)
	t.Logf("BetaAnthropic: team=%s owner=%s admin=%s member=%s",
		identity.BetaAnthropicTeamID, identity.BetaAnthropicOwnerUserID, identity.BetaAnthropicAdminUserID, identity.BetaAnthropicMemberUserID)
	t.Logf("BetaHaiku:     team=%s owner=%s admin=%s member=%s",
		identity.BetaHaikuTeamID, identity.BetaHaikuOwnerUserID, identity.BetaHaikuAdminUserID, identity.BetaHaikuMemberUserID)
	t.Logf("BetaGPT4o:     team=%s owner=%s admin=%s member=%s",
		identity.BetaGPT4oTeamID, identity.BetaGPT4oOwnerUserID, identity.BetaGPT4oAdminUserID, identity.BetaGPT4oMemberUserID)
}

func TestGenerateAppEnv_FiltersOrchestrationKeys(t *testing.T) {
	pairs := map[string]string{
		"YAO_DB_DRIVER":    "sqlite3",
		"MOCK_LLM_HOST":    "http://test-mock:1234",
		"OPENAI_API_KEY":   "sk-test",
		"TEST_FOO":         "should-not-appear",
		"SANDBOX_TEST_IMG": "should-not-appear",
	}

	err := generateAppEnv(pairs)
	if err != nil {
		t.Fatalf("generateAppEnv: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(agentAppDir, ".env"))
	if err != nil {
		t.Fatalf("read app/.env: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "YAO_DB_DRIVER=sqlite3") {
		t.Error("missing YAO_DB_DRIVER=sqlite3")
	}
	if !strings.Contains(content, "MOCK_LLM_HOST=") {
		t.Error("missing MOCK_LLM_HOST")
	}
	if !strings.Contains(content, "OPENAI_API_KEY=sk-test") {
		t.Error("missing OPENAI_API_KEY")
	}
	if strings.Contains(content, "TEST_FOO") {
		t.Error("app/.env should not contain TEST_FOO")
	}
	if strings.Contains(content, "SANDBOX_TEST_IMG") {
		t.Error("app/.env should not contain SANDBOX_TEST_IMG")
	}
}
