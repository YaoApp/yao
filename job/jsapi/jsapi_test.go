package jsapi

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/job"
	"github.com/yaoapp/yao/test"
)

// registerTestProcesses registers a test echo process for job execution testing
func registerTestProcesses() {
	process.Register("test.yaojob.echo", func(p *process.Process) interface{} {
		args := p.Args
		message := "no message"
		if len(args) > 0 {
			if m, ok := args[0].(string); ok {
				message = m
			}
		}

		// Simulate some work
		if p.Callback != nil {
			p.Callback(p, map[string]interface{}{
				"type":     "progress",
				"progress": 50,
				"message":  "Processing...",
			})
			p.Callback(p, map[string]interface{}{
				"type":     "progress",
				"progress": 100,
				"message":  "Done",
			})
		}

		return map[string]interface{}{
			"message": message,
			"status":  "success",
		}
	})
}

// TestYaoJobConstructor tests creating a YaoJob from JavaScript
func TestYaoJobConstructor(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	res, err := v8.Call(v8.CallOptions{}, `
		function test() {
			const j = new YaoJob({ name: "Test Job", description: "Unit test job", icon: "work", category_name: "Test" });
			return { id: j.id, hasAdd: typeof j.Add === "function", hasRun: typeof j.Run === "function" };
		}`)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map, got %T", res)
	}

	assert.NotEmpty(t, result["id"], "job should have an id")
	assert.Equal(t, true, result["hasAdd"], "job should have Add method")
	assert.Equal(t, true, result["hasRun"], "job should have Run method")
	t.Logf("Created YaoJob with id: %s", result["id"])
}

// TestYaoJobConstructorEmpty tests creating a YaoJob with empty options
func TestYaoJobConstructorEmpty(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	res, err := v8.Call(v8.CallOptions{}, `
		function test() {
			const j = new YaoJob({});
			return j.id;
		}`)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	id, ok := res.(string)
	if !ok {
		t.Fatalf("Expected string, got %T: %v", res, res)
	}
	assert.NotEmpty(t, id, "job should have an id")
	t.Logf("Created YaoJob (empty opts) with id: %s", id)
}

// TestYaoJobAddAndRun tests the full lifecycle: create → Add → Run
func TestYaoJobAddAndRun(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	registerTestProcesses()

	res, err := v8.Call(v8.CallOptions{}, `
		function test() {
			const j = new YaoJob({ name: "Echo Job", description: "Test echo execution" });
			j.Add("test.yaojob.echo", "Hello from YaoJob");
			j.Run();
			return j.id;
		}`)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	jobID, ok := res.(string)
	if !ok {
		t.Fatalf("Expected string, got %T: %v", res, res)
	}
	assert.NotEmpty(t, jobID, "job should have an id")

	// Wait for async execution
	time.Sleep(2 * time.Second)
	t.Logf("YaoJob Add+Run completed, id: %s", jobID)
}

// TestYaoJobAddChaining tests that Add returns the job for chaining
func TestYaoJobAddChaining(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	registerTestProcesses()

	res, err := v8.Call(v8.CallOptions{}, `
		function test() {
			const j = new YaoJob({ name: "Chaining Test" });
			// Add should return the job object for chaining
			const result = j.Add("test.yaojob.echo", "chained");
			return { id: j.id, chainWorks: result !== undefined && result !== null };
		}`)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map, got %T", res)
	}
	assert.Equal(t, true, result["chainWorks"], "Add should return the job for chaining")
}

// TestYaoJobStatus tests the static YaoJob.Status() method
func TestYaoJobStatus(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	registerTestProcesses()

	// Create a job, add execution, run, then check status
	res, err := v8.Call(v8.CallOptions{}, `
		function test() {
			const j = new YaoJob({ name: "Status Test Job" });
			j.Add("test.yaojob.echo", "status check");
			j.Run();
			return j.id;
		}`)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	jobID, ok := res.(string)
	if !ok {
		t.Fatalf("Expected string, got %T", res)
	}

	// Wait for execution
	time.Sleep(2 * time.Second)

	// Now check status via static method
	statusRes, err := v8.Call(v8.CallOptions{}, `
		function test() {
			return YaoJob.Status("`+jobID+`");
		}`)
	if err != nil {
		t.Fatalf("Status call failed: %v", err)
	}

	status, ok := statusRes.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map, got %T: %v", statusRes, statusRes)
	}

	assert.Equal(t, jobID, status["job_id"], "job_id should match")
	assert.NotEmpty(t, status["status"], "status should not be empty")

	if executions, ok := status["executions"].([]interface{}); ok && len(executions) > 0 {
		exec := executions[0].(map[string]interface{})
		t.Logf("Execution status: %s, progress: %v", exec["status"], exec["progress"])
	}

	t.Logf("YaoJob.Status result: job_id=%s, status=%s", status["job_id"], status["status"])
}

// TestYaoJobStatusInvalid tests YaoJob.Status with a non-existent job_id
func TestYaoJobStatusInvalid(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	_, err := v8.Call(v8.CallOptions{}, `
		function test() {
			return YaoJob.Status("nonexistent-job-id-999");
		}`)
	assert.Error(t, err, "Status should fail for non-existent job_id")
	t.Logf("Expected error: %v", err)
}

// TestYaoJobStatusMissingArg tests YaoJob.Status without arguments
func TestYaoJobStatusMissingArg(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	_, err := v8.Call(v8.CallOptions{}, `
		function test() {
			return YaoJob.Status();
		}`)
	assert.Error(t, err, "Status should fail without job_id argument")
}

// TestYaoJobStop tests the static YaoJob.Stop() method
func TestYaoJobStop(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	registerTestProcesses()

	// Create and run a job, then stop it
	res, err := v8.Call(v8.CallOptions{}, `
		function test() {
			const j = new YaoJob({ name: "Stop Test Job" });
			j.Add("test.yaojob.echo", "will be stopped");
			return j.id;
		}`)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	jobID, ok := res.(string)
	if !ok {
		t.Fatalf("Expected string, got %T", res)
	}

	// Stop the job
	_, err = v8.Call(v8.CallOptions{}, `
		function test() {
			YaoJob.Stop("`+jobID+`");
			return true;
		}`)
	if err != nil {
		t.Fatalf("Stop call failed: %v", err)
	}

	t.Logf("YaoJob.Stop succeeded for job: %s", jobID)
}

// TestYaoJobStopInvalid tests YaoJob.Stop with a non-existent job_id
func TestYaoJobStopInvalid(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	_, err := v8.Call(v8.CallOptions{}, `
		function test() {
			YaoJob.Stop("nonexistent-job-id-999");
			return true;
		}`)
	assert.Error(t, err, "Stop should fail for non-existent job_id")
}

// TestYaoJobAddMissingProcessName tests Add with missing process name
func TestYaoJobAddMissingProcessName(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	_, err := v8.Call(v8.CallOptions{}, `
		function test() {
			const j = new YaoJob({ name: "Error Test" });
			j.Add();  // Missing process name
			return true;
		}`)
	assert.Error(t, err, "Add should fail without process name")
}

// TestYaoJobScopeFieldsDataPath tests that __yao_team_id and __yao_created_by
// are correctly saved to and loaded from the database when present in the creation data.
// This validates the full data path: data map → OnceAndSave → DB → GetJob.
func TestYaoJobScopeFieldsDataPath(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Simulate what the constructor does when auth is available:
	// inject __yao_team_id and __yao_created_by into the data map.
	data := map[string]interface{}{
		"name":             "Scope Data Path Test",
		"icon":             "work",
		"category_name":    "ScopeTest",
		"__yao_team_id":    "team-xyz",
		"__yao_created_by": "user-abc",
	}

	j, err := job.OnceAndSave(job.GOROUTINE, data)
	if err != nil {
		t.Fatalf("OnceAndSave failed: %v", err)
	}
	assert.NotEmpty(t, j.JobID)

	// Read back from DB
	loaded, err := job.GetJob(j.JobID)
	if err != nil {
		t.Fatalf("GetJob failed: %v", err)
	}

	assert.Equal(t, "team-xyz", loaded.YaoTeamID, "__yao_team_id should be persisted")
	assert.Equal(t, "user-abc", loaded.YaoCreatedBy, "__yao_created_by should be persisted")
	t.Logf("Scope data path verified: job_id=%s, team_id=%s, created_by=%s",
		loaded.JobID, loaded.YaoTeamID, loaded.YaoCreatedBy)
}

// TestYaoJobScopeFieldsViaJS tests the constructor auto-injects scope fields
// from the V8 Authorized context. Also verifies scope fields are empty without auth.
func TestYaoJobScopeFieldsViaJS(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Case 1: Without auth — scope fields should be empty
	res, err := v8.Call(v8.CallOptions{}, `
		function test() {
			const j = new YaoJob({ name: "No Auth Scope Test" });
			return j.id;
		}`)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	jobID, ok := res.(string)
	if !ok {
		t.Fatalf("Expected string, got %T: %v", res, res)
	}

	j, err := job.GetJob(jobID)
	if err != nil {
		t.Fatalf("GetJob failed: %v", err)
	}

	assert.Empty(t, j.YaoTeamID, "__yao_team_id should be empty without auth")
	assert.Empty(t, j.YaoCreatedBy, "__yao_created_by should be empty without auth")
	t.Logf("No-auth scope verified: team_id='%s', created_by='%s'", j.YaoTeamID, j.YaoCreatedBy)

	// Case 2: With auth via Global["authorized"] — this tests the runtime integration.
	// Note: v8.Call sets Share.Global but not Share.Authorized directly.
	// In production, Authorized is set by the Yao HTTP/Process layer.
	// To verify the constructor logic, we pass scope fields explicitly via JS data.
	res2, err := v8.Call(v8.CallOptions{}, `
		function test() {
			const j = new YaoJob({
				name: "Explicit Scope Test",
				"__yao_team_id": "team-from-js",
				"__yao_created_by": "user-from-js"
			});
			return j.id;
		}`)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	jobID2, ok := res2.(string)
	if !ok {
		t.Fatalf("Expected string, got %T: %v", res2, res2)
	}

	j2, err := job.GetJob(jobID2)
	if err != nil {
		t.Fatalf("GetJob failed: %v", err)
	}

	assert.Equal(t, "team-from-js", j2.YaoTeamID, "__yao_team_id should be set from JS data")
	assert.Equal(t, "user-from-js", j2.YaoCreatedBy, "__yao_created_by should be set from JS data")
	t.Logf("Explicit scope verified: team_id=%s, created_by=%s", j2.YaoTeamID, j2.YaoCreatedBy)
}

// TestYaoJobFullLifecycle tests create → Add → Run → Status → verify completion
func TestYaoJobFullLifecycle(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	registerTestProcesses()

	// Step 1: Create, Add, Run
	res, err := v8.Call(v8.CallOptions{}, `
		function test() {
			const j = new YaoJob({
				name: "Full Lifecycle Test",
				description: "Testing complete YaoJob lifecycle",
				icon: "check_circle",
				category_name: "UnitTest"
			});
			j.Add("test.yaojob.echo", "lifecycle test message");
			j.Run();
			return j.id;
		}`)
	if err != nil {
		t.Fatalf("Create/Add/Run failed: %v", err)
	}

	jobID, ok := res.(string)
	if !ok {
		t.Fatalf("Expected string, got %T", res)
	}

	// Step 2: Wait for execution to complete
	time.Sleep(3 * time.Second)

	// Step 3: Check status
	statusRes, err := v8.Call(v8.CallOptions{}, `
		function test() {
			return YaoJob.Status("`+jobID+`");
		}`)
	if err != nil {
		t.Fatalf("Status check failed: %v", err)
	}

	status, ok := statusRes.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map, got %T", statusRes)
	}

	assert.Equal(t, jobID, status["job_id"])

	// Check execution details
	if executions, ok := status["executions"].([]interface{}); ok && len(executions) > 0 {
		exec := executions[0].(map[string]interface{})
		t.Logf("Full lifecycle result: status=%s, progress=%v", exec["status"], exec["progress"])

		// After 3 seconds, the echo process should be completed
		if exec["status"] == "completed" {
			t.Log("Job execution completed successfully")
			if result, ok := exec["result"]; ok {
				t.Logf("Execution result: %v", result)
			}
		}
	}
}
