package jsapi

import (
	"fmt"

	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/yao/job"
	"rogchap.com/v8go"
)

func init() {
	v8.RegisterFunction("YaoJob", ExportFunction)
}

// ExportFunction exports the YaoJob constructor function template.
//
// Usage from JavaScript:
//
//	// Create a persistent Job
//	const j = new YaoJob({ name: "Fetch webpage", icon: "language", category_name: "Keeper" });
//	j.Add("agents.yao.keeper.webfetch.URL", teamId, url, opts);
//	j.Run();
//	const jobId = j.id;
//
//	// Static methods (no instance needed)
//	const status = YaoJob.Status("job-id-xxx");
//	YaoJob.Stop("job-id-xxx");
func ExportFunction(iso *v8go.Isolate) *v8go.FunctionTemplate {
	tmpl := v8go.NewFunctionTemplate(iso, yaoJobConstructor)

	// Register static methods on the constructor function itself
	tmpl.Set("Status", yaoJobStatusStatic(iso))
	tmpl.Set("Stop", yaoJobStopStatic(iso))

	return tmpl
}

// yaoJobConstructor is the JavaScript constructor for YaoJob.
// Usage: new YaoJob({ name: "...", icon: "...", description: "...", category_name: "..." })
//
// Internally calls job.OnceAndSave("GOROUTINE", data).
// The JS object only stores job_id as a string — no Go pointer held.
func yaoJobConstructor(info *v8go.FunctionCallbackInfo) *v8go.Value {
	ctx := info.Context()
	iso := ctx.Isolate()
	args := info.Args()

	// Parse data argument
	data := make(map[string]interface{})
	if len(args) > 0 && !args[0].IsNullOrUndefined() {
		goVal, err := bridge.GoValue(args[0], ctx)
		if err != nil {
			return bridge.JsException(ctx, fmt.Sprintf("YaoJob: invalid argument: %s", err))
		}
		if m, ok := goVal.(map[string]interface{}); ok {
			data = m
		}
	}

	// Capture current V8 context's auth info to populate scope fields
	if share, err := bridge.ShareData(ctx); err == nil && share != nil {
		if share.Authorized != nil {
			if teamID, ok := share.Authorized["team_id"].(string); ok && teamID != "" {
				data["__yao_team_id"] = teamID
			}
			if userID, ok := share.Authorized["user_id"].(string); ok && userID != "" {
				data["__yao_created_by"] = userID
			}
		}
	}

	// Create and persist job
	j, err := job.OnceAndSave(job.GOROUTINE, data)
	if err != nil {
		return bridge.JsException(ctx, fmt.Sprintf("YaoJob: failed to create job: %s", err))
	}

	// Build the JS instance object — only stores job_id string
	objTmpl := v8go.NewObjectTemplate(iso)
	objTmpl.Set("id", j.JobID)
	objTmpl.Set("Add", yaoJobAddMethod(iso, j.JobID))
	objTmpl.Set("Run", yaoJobRunMethod(iso, j.JobID))

	instance, err := objTmpl.NewInstance(ctx)
	if err != nil {
		return bridge.JsException(ctx, fmt.Sprintf("YaoJob: failed to create instance: %s", err))
	}

	return instance.Value
}

// yaoJobAddMethod creates the Add instance method.
// Usage: job.Add("processName", arg1, arg2, ...)
//
// Loads *Job from DB by job_id, calls job.Add(options, processName, args...), then *Job is discarded.
// Automatically captures the current V8 context's Sid and Authorized info into
// ExecutionOptions.SharedData, so the Job Worker can restore them when executing.
func yaoJobAddMethod(iso *v8go.Isolate, jobID string) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		ctx := info.Context()
		args := info.Args()

		if len(args) < 1 || !args[0].IsString() {
			return bridge.JsException(ctx, "YaoJob.Add: first argument must be a process name string")
		}

		processName := args[0].String()

		// Convert remaining JS args to Go values
		var processArgs []interface{}
		if len(args) > 1 {
			goArgs, err := bridge.GoValues(args[1:], ctx)
			if err != nil {
				return bridge.JsException(ctx, fmt.Sprintf("YaoJob.Add: invalid arguments: %s", err))
			}
			processArgs = goArgs
		}

		// Capture current V8 context's auth info for the Job Worker
		opts := &job.ExecutionOptions{
			SharedData: make(map[string]interface{}),
		}
		if share, err := bridge.ShareData(ctx); err == nil && share != nil {
			if share.Sid != "" {
				opts.SharedData["sid"] = share.Sid
			}
			if share.Authorized != nil {
				opts.SharedData["authorized"] = share.Authorized
			}
		}

		// Load job from DB (stateless — no Go pointer held)
		j, err := job.GetJob(jobID)
		if err != nil {
			return bridge.JsException(ctx, fmt.Sprintf("YaoJob.Add: failed to load job %s: %s", jobID, err))
		}

		// Add execution with auth context
		if err := j.Add(opts, processName, processArgs...); err != nil {
			return bridge.JsException(ctx, fmt.Sprintf("YaoJob.Add: failed to add execution: %s", err))
		}

		// Return this for chaining
		return info.This().Value
	})
}

// yaoJobRunMethod creates the Run instance method.
// Usage: job.Run()
//
// Loads *Job from DB by job_id, calls job.Push() to submit to worker queue, then *Job is discarded.
func yaoJobRunMethod(iso *v8go.Isolate, jobID string) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		ctx := info.Context()

		// Load job from DB (stateless)
		j, err := job.GetJob(jobID)
		if err != nil {
			return bridge.JsException(ctx, fmt.Sprintf("YaoJob.Run: failed to load job %s: %s", jobID, err))
		}

		// Push to worker queue (async execution)
		if err := j.Push(); err != nil {
			return bridge.JsException(ctx, fmt.Sprintf("YaoJob.Run: failed to run job: %s", err))
		}

		return v8go.Undefined(iso)
	})
}

// yaoJobStatusStatic creates the static YaoJob.Status(jobId) method.
// Usage: YaoJob.Status("job-id-xxx")
//
// Returns: { job_id, status, executions: [{ execution_id, status, progress, result?, error? }] }
func yaoJobStatusStatic(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		ctx := info.Context()
		args := info.Args()

		if len(args) < 1 || !args[0].IsString() {
			return bridge.JsException(ctx, "YaoJob.Status: job_id (string) is required")
		}

		jobID := args[0].String()

		// Load job
		j, err := job.GetJob(jobID)
		if err != nil {
			return bridge.JsException(ctx, fmt.Sprintf("YaoJob.Status: failed to load job %s: %s", jobID, err))
		}

		// Load executions
		executions, err := job.GetExecutions(jobID)
		if err != nil {
			return bridge.JsException(ctx, fmt.Sprintf("YaoJob.Status: failed to load executions: %s", err))
		}

		// Build result
		execList := make([]interface{}, 0, len(executions))
		for _, exec := range executions {
			entry := map[string]interface{}{
				"execution_id": exec.ExecutionID,
				"status":       exec.Status,
				"progress":     exec.Progress,
			}
			if exec.Result != nil {
				entry["result"] = string(*exec.Result)
			}
			if exec.ErrorInfo != nil {
				entry["error"] = string(*exec.ErrorInfo)
			}
			execList = append(execList, entry)
		}

		result := map[string]interface{}{
			"job_id":     j.JobID,
			"status":     j.Status,
			"executions": execList,
		}

		jsVal, err := bridge.JsValue(ctx, result)
		if err != nil {
			return bridge.JsException(ctx, fmt.Sprintf("YaoJob.Status: failed to convert result: %s", err))
		}

		return jsVal
	})
}

// yaoJobStopStatic creates the static YaoJob.Stop(jobId) method.
// Usage: YaoJob.Stop("job-id-xxx")
func yaoJobStopStatic(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		ctx := info.Context()
		args := info.Args()

		if len(args) < 1 || !args[0].IsString() {
			return bridge.JsException(ctx, "YaoJob.Stop: job_id (string) is required")
		}

		jobID := args[0].String()

		// Load job from DB
		j, err := job.GetJob(jobID)
		if err != nil {
			return bridge.JsException(ctx, fmt.Sprintf("YaoJob.Stop: failed to load job %s: %s", jobID, err))
		}

		// Stop the job
		if err := j.Stop(); err != nil {
			return bridge.JsException(ctx, fmt.Sprintf("YaoJob.Stop: failed to stop job: %s", err))
		}

		return v8go.Undefined(iso)
	})
}
