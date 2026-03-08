package jsapi

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	sandbox "github.com/yaoapp/yao/sandbox/v2"
	wsjsapi "github.com/yaoapp/yao/workspace/jsapi"
	"rogchap.com/v8go"
)

// ---------------------------------------------------------------------------
// Helpers — shared across jsapi files
// ---------------------------------------------------------------------------

func throwError(info *v8go.FunctionCallbackInfo, msg string) *v8go.Value {
	iso := info.Context().Isolate()
	e, _ := v8go.NewValue(iso, msg)
	iso.ThrowException(e)
	return v8go.Undefined(iso)
}

func parseStringArray(val *v8go.Value) []string {
	obj, err := val.AsObject()
	if err != nil {
		return nil
	}
	lenVal, err := obj.Get("length")
	if err != nil {
		return nil
	}
	length := int(lenVal.Int32())
	result := make([]string, 0, length)
	for i := 0; i < length; i++ {
		item, err := obj.GetIdx(uint32(i))
		if err != nil || !item.IsString() {
			continue
		}
		result = append(result, item.String())
	}
	return result
}

func parseStringMap(v8ctx *v8go.Context, val *v8go.Value) map[string]string {
	result := make(map[string]string)
	if !val.IsObject() {
		return result
	}
	jsonStr, err := v8go.JSONStringify(v8ctx, val)
	if err != nil {
		return result
	}
	_ = json.Unmarshal([]byte(jsonStr), &result)
	return result
}

func parseExecOptions(v8ctx *v8go.Context, args []*v8go.Value) ([]string, []sandbox.ExecOption, *v8go.Value) {
	if len(args) < 1 || !args[0].IsObject() {
		return nil, nil, nil
	}
	cmd := parseStringArray(args[0])
	if len(cmd) == 0 {
		return nil, nil, nil
	}
	var opts []sandbox.ExecOption
	var callback *v8go.Value
	for i := 1; i < len(args); i++ {
		v := args[i]
		if v.IsFunction() {
			callback = v
			break
		}
		if v.IsObject() {
			optsObj, err := v.AsObject()
			if err != nil {
				continue
			}
			if wd, e := optsObj.Get("workdir"); e == nil && wd.IsString() {
				opts = append(opts, sandbox.WithWorkDir(wd.String()))
			}
			if env, e := optsObj.Get("env"); e == nil && env.IsObject() {
				envMap := parseStringMap(v8ctx, env)
				if len(envMap) > 0 {
					opts = append(opts, sandbox.WithEnv(envMap))
				}
			}
			if stdin, e := optsObj.Get("stdin"); e == nil && stdin.IsString() {
				opts = append(opts, sandbox.WithStdin([]byte(stdin.String())))
			}
			if t, e := optsObj.Get("timeout"); e == nil && t.IsNumber() {
				opts = append(opts, sandbox.WithTimeout(time.Duration(t.Number())*time.Millisecond))
			}
			if mo, e := optsObj.Get("max_output"); e == nil && mo.IsNumber() {
				opts = append(opts, sandbox.WithMaxOutput(int64(mo.Number())))
			}
		}
	}
	return cmd, opts, callback
}

func execResultToJS(v8ctx *v8go.Context, r *sandbox.ExecResult) *v8go.Value {
	data, _ := json.Marshal(map[string]interface{}{
		"exit_code":   r.ExitCode,
		"stdout":      r.Stdout,
		"stderr":      r.Stderr,
		"duration_ms": r.DurationMs,
		"error":       r.Error,
		"truncated":   r.Truncated,
	})
	val, _ := v8go.JSONParse(v8ctx, string(data))
	return val
}

func boxInfoToJS(v8ctx *v8go.Context, b *sandbox.BoxInfo) *v8go.Value {
	data, _ := json.Marshal(map[string]interface{}{
		"id":            b.ID,
		"container_id":  b.ContainerID,
		"node_id":       b.NodeID,
		"owner":         b.Owner,
		"status":        b.Status,
		"image":         b.Image,
		"vnc":           b.VNC,
		"policy":        string(b.Policy),
		"labels":        b.Labels,
		"created_at":    b.CreatedAt.Format(time.RFC3339),
		"last_active":   b.LastActive.Format(time.RFC3339),
		"process_count": b.ProcessCount,
	})
	val, _ := v8go.JSONParse(v8ctx, string(data))
	return val
}

func computerInfoToJS(v8ctx *v8go.Context, c sandbox.ComputerInfo) *v8go.Value {
	data, _ := json.Marshal(map[string]interface{}{
		"kind":         c.Kind,
		"node_id":      c.NodeID,
		"tai_id":       c.TaiID,
		"machine_id":   c.MachineID,
		"version":      c.Version,
		"mode":         c.Mode,
		"status":       c.Status,
		"capabilities": c.Capabilities,
		"system": map[string]interface{}{
			"os":        c.System.OS,
			"arch":      c.System.Arch,
			"hostname":  c.System.Hostname,
			"num_cpu":   c.System.NumCPU,
			"total_mem": c.System.TotalMem,
		},
		"box_id":       c.BoxID,
		"container_id": c.ContainerID,
		"owner":        c.Owner,
		"image":        c.Image,
		"policy":       string(c.Policy),
		"labels":       c.Labels,
	})
	val, _ := v8go.JSONParse(v8ctx, string(data))
	return val
}

// getComputer re-fetches a Computer from the Manager by kind + identifier.
// kind="box"  → identifier is boxID, kind="host" → identifier is node ID.
func getComputer(ctx context.Context, kind, identifier string) (sandbox.Computer, error) {
	m := sandbox.M()
	if kind == "box" {
		return m.Get(ctx, identifier)
	}
	return m.Host(ctx, identifier)
}

// ---------------------------------------------------------------------------
// sbHost — sandbox.Host(nodeID?)
// ---------------------------------------------------------------------------

func sbHost(info *v8go.FunctionCallbackInfo) *v8go.Value {
	ctx := context.Background()
	v8ctx := info.Context()

	nodeID := ""
	args := info.Args()
	if len(args) > 0 && args[0].IsString() {
		nodeID = args[0].String()
	}

	if _, err := sandbox.M().Host(ctx, nodeID); err != nil {
		return throwError(info, err.Error())
	}

	val, err := NewComputerObject(v8ctx, "host", nodeID)
	if err != nil {
		return throwError(info, err.Error())
	}
	return val
}

// ---------------------------------------------------------------------------
// NewComputerObject — unified JS Computer object factory
// ---------------------------------------------------------------------------

// NewComputerObject creates a JS Computer object. Closures capture only
// kind (string) and identifier (string) — no Go objects cross into V8.
func NewComputerObject(v8ctx *v8go.Context, kind string, identifier string) (*v8go.Value, error) {
	iso := v8ctx.Isolate()
	ctx := context.Background()

	// Mutable workplace binding lives in closure, not in V8 heap.
	var workplaceID string

	tpl := v8go.NewObjectTemplate(iso)

	// -- Exec --
	tpl.Set("Exec", v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		cmd, opts, _ := parseExecOptions(info.Context(), info.Args())
		if len(cmd) == 0 {
			return throwError(info, "Exec requires cmd (string[])")
		}
		comp, err := getComputer(ctx, kind, identifier)
		if err != nil {
			return throwError(info, err.Error())
		}
		result, err := comp.Exec(ctx, cmd, opts...)
		if err != nil {
			return throwError(info, err.Error())
		}
		return execResultToJS(info.Context(), result)
	}))

	// -- Stream --
	tpl.Set("Stream", v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		cmd, opts, cbVal := parseExecOptions(info.Context(), info.Args())
		if len(cmd) == 0 {
			return throwError(info, "Stream requires cmd (string[]) and callback")
		}
		if cbVal == nil || !cbVal.IsFunction() {
			return throwError(info, "Stream requires a callback function as last argument")
		}
		cbFn, err := cbVal.AsFunction()
		if err != nil {
			return throwError(info, "Stream callback is not a function")
		}
		comp, err := getComputer(ctx, kind, identifier)
		if err != nil {
			return throwError(info, err.Error())
		}
		stream, err := comp.Stream(ctx, cmd, opts...)
		if err != nil {
			return throwError(info, err.Error())
		}

		type chunk struct {
			typ  string
			data interface{}
		}
		ch := make(chan chunk, 64)
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			buf := make([]byte, 4096)
			for {
				n, err := stream.Stdout.Read(buf)
				if n > 0 {
					ch <- chunk{"stdout", string(buf[:n])}
				}
				if err != nil {
					break
				}
			}
		}()
		go func() {
			defer wg.Done()
			buf := make([]byte, 4096)
			for {
				n, err := stream.Stderr.Read(buf)
				if n > 0 {
					ch <- chunk{"stderr", string(buf[:n])}
				}
				if err != nil {
					break
				}
			}
		}()
		go func() {
			code, _ := stream.Wait()
			wg.Wait()
			ch <- chunk{"exit", code}
			close(ch)
		}()

		v8c := info.Context()
		global := v8c.Global()
		for c := range ch {
			var dataVal *v8go.Value
			switch v := c.data.(type) {
			case string:
				dataVal, _ = v8go.NewValue(iso, v)
			case int:
				dataVal, _ = v8go.NewValue(iso, int32(v))
			}
			typeVal, _ := v8go.NewValue(iso, c.typ)
			if typeVal != nil && dataVal != nil {
				_, _ = cbFn.Call(global, typeVal, dataVal)
			}
		}
		return v8go.Undefined(iso)
	}))

	// -- VNC --
	tpl.Set("VNC", v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		comp, err := getComputer(ctx, kind, identifier)
		if err != nil {
			return throwError(info, err.Error())
		}
		url, err := comp.VNC(ctx)
		if err != nil {
			return throwError(info, err.Error())
		}
		val, _ := v8go.NewValue(iso, url)
		return val
	}))

	// -- Proxy --
	tpl.Set("Proxy", v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 || !args[0].IsNumber() {
			return throwError(info, "Proxy requires port (number)")
		}
		port := int(args[0].Int32())
		path := "/"
		if len(args) > 1 && args[1].IsString() {
			path = args[1].String()
		}
		comp, err := getComputer(ctx, kind, identifier)
		if err != nil {
			return throwError(info, err.Error())
		}
		url, err := comp.Proxy(ctx, port, path)
		if err != nil {
			return throwError(info, err.Error())
		}
		val, _ := v8go.NewValue(iso, url)
		return val
	}))

	// -- ComputerInfo --
	tpl.Set("ComputerInfo", v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		comp, err := getComputer(ctx, kind, identifier)
		if err != nil {
			return throwError(info, err.Error())
		}
		return computerInfoToJS(info.Context(), comp.ComputerInfo())
	}))

	// -- BindWorkplace --
	tpl.Set("BindWorkplace", v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 || !args[0].IsString() {
			return throwError(info, "BindWorkplace requires workspaceID (string)")
		}
		workplaceID = args[0].String()
		comp, err := getComputer(ctx, kind, identifier)
		if err != nil {
			return throwError(info, err.Error())
		}
		comp.BindWorkplace(workplaceID)
		return v8go.Undefined(iso)
	}))

	// -- Workplace → reuse workspace JSAPI NewFSObject --
	tpl.Set("Workplace", v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		if workplaceID == "" {
			return v8go.Null(iso)
		}
		val, err := wsjsapi.NewFSObject(info.Context(), workplaceID)
		if err != nil {
			return throwError(info, err.Error())
		}
		return val
	}))

	// -- Box-only: Info --
	tpl.Set("Info", v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		if kind == "host" {
			return throwError(info, "not supported: Info() requires a box computer")
		}
		comp, err := getComputer(ctx, kind, identifier)
		if err != nil {
			return throwError(info, err.Error())
		}
		box := comp.(*sandbox.Box)
		bi, err := box.Info(ctx)
		if err != nil {
			return throwError(info, err.Error())
		}
		return boxInfoToJS(info.Context(), bi)
	}))

	// -- Box-only: Start --
	tpl.Set("Start", v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		if kind == "host" {
			return throwError(info, "not supported: Start() requires a box computer")
		}
		comp, err := getComputer(ctx, kind, identifier)
		if err != nil {
			return throwError(info, err.Error())
		}
		if err := comp.(*sandbox.Box).Start(ctx); err != nil {
			return throwError(info, err.Error())
		}
		return v8go.Undefined(iso)
	}))

	// -- Box-only: Stop --
	tpl.Set("Stop", v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		if kind == "host" {
			return throwError(info, "not supported: Stop() requires a box computer")
		}
		comp, err := getComputer(ctx, kind, identifier)
		if err != nil {
			return throwError(info, err.Error())
		}
		if err := comp.(*sandbox.Box).Stop(ctx); err != nil {
			return throwError(info, err.Error())
		}
		return v8go.Undefined(iso)
	}))

	// -- Box-only: Remove --
	tpl.Set("Remove", v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		if kind == "host" {
			return throwError(info, "not supported: Remove() requires a box computer")
		}
		comp, err := getComputer(ctx, kind, identifier)
		if err != nil {
			return throwError(info, err.Error())
		}
		if err := comp.(*sandbox.Box).Remove(ctx); err != nil {
			return throwError(info, err.Error())
		}
		return v8go.Undefined(iso)
	}))

	// Instantiate and set read-only properties
	obj, err := tpl.NewInstance(v8ctx)
	if err != nil {
		return nil, err
	}

	obj.Set("kind", kind)

	idStr := ""
	ownerStr := ""
	nodeIDStr := identifier
	if kind == "box" {
		if comp, err := getComputer(ctx, kind, identifier); err == nil {
			box := comp.(*sandbox.Box)
			idStr = box.ID()
			ownerStr = box.Owner()
			nodeIDStr = box.NodeID()
		} else {
			idStr = identifier
		}
	}
	obj.Set("id", idStr)
	obj.Set("owner", ownerStr)
	obj.Set("node_id", nodeIDStr)

	return obj.Value, nil
}
