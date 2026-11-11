//go:build e2e

package computer

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/e2e/env"
)

var (
	sharedEnv *env.Environment
	sharedErr error
)

// TestMain creates one isolated environment for the whole package and closes
// it after all tests finish.
func TestMain(m *testing.M) {
	code := m.Run()
	if sharedEnv != nil {
		sharedEnv.Close()
	}
	os.Exit(code)
}

func testEnvironment(t *testing.T) *env.Environment {
	t.Helper()
	if sharedEnv == nil {
		sharedEnv, sharedErr = env.NewShared(t)
	}
	if sharedErr != nil {
		t.Fatalf("create E2E environment: %v", sharedErr)
	}
	return sharedEnv
}

// TestE2EEnvironmentSmoke verifies the stage-1 E2E environment end to end.
func TestE2EEnvironmentSmoke(t *testing.T) {
	environment := testEnvironment(t)

	if environment.DB.Name() == "" {
		t.Fatal("database name is empty")
	}
	if !strings.Contains(environment.DB.Name(), "yao_e2e") && environment.DB.Driver() == "postgres" {
		t.Fatalf("unexpected postgres test database: %s", environment.DB.Name())
	}
	if environment.TaiBin == "" || environment.TaiVersion == "" {
		t.Fatal("Tai binary/version is empty")
	}

	resp, body, err := environment.Get("/v1/oauth/jwks")
	if err != nil {
		t.Fatalf("GET jwks: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET jwks status=%d body=%s", resp.StatusCode, body)
	}

	resp, body, err = environment.Get("/v1/user/entry?locale=en")
	if err != nil {
		t.Fatalf("GET entry: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET entry status=%d body=%s", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), "Yao Agents E2E") {
		t.Fatalf("entry response missing app title: %s", body)
	}

	runJSON := filepath.Join(environment.RunDir, "run.json")
	if _, err := os.Stat(runJSON); err != nil {
		t.Fatalf("run.json missing: %v", err)
	}

	verifyComputerWriteReadDelete(t)
}

// verifyComputerWriteReadDelete exercises the real computer table in the
// isolated test database.
func verifyComputerWriteReadDelete(t *testing.T) {
	t.Helper()

	computerModel := model.Select("__yao.computers")
	if computerModel == nil {
		t.Fatal("computer model not loaded")
	}

	id := fmt.Sprintf("e2e-%d", time.Now().UnixNano())
	row := maps.MapStrAny{
		"id":            id,
		"node_id":       "node-" + id,
		"name":          "E2E Smoke Computer",
		"provider_type": "local",
		"scope":         "system",
		"status":        "offline",
	}

	columns := []string{"id", "node_id", "name", "provider_type", "scope", "status"}
	values := [][]interface{}{{row["id"], row["node_id"], row["name"], row["provider_type"], row["scope"], row["status"]}}
	if err := computerModel.Insert(columns, values); err != nil {
		t.Fatalf("insert computer: %v", err)
	}
	t.Cleanup(func() { _ = computerModel.Delete(id) })

	found, err := computerModel.Find(id, model.QueryParam{})
	if err != nil {
		t.Fatalf("find computer: %v", err)
	}
	if name, _ := found["name"].(string); name != "E2E Smoke Computer" {
		t.Fatalf("unexpected computer name: %v", found["name"])
	}
	if nodeID, _ := found["node_id"].(string); nodeID != "node-"+id {
		t.Fatalf("unexpected computer node_id: %v", found["node_id"])
	}

	if err := computerModel.Delete(id); err != nil {
		t.Fatalf("delete computer: %v", err)
	}
	if _, err := computerModel.Find(id, model.QueryParam{}); err == nil {
		t.Fatal("computer still exists after delete")
	}
}
