// Package jsapi registers the sandbox namespace into the Yao V8 runtime.
//
// All methods are static on the sandbox object — no constructor.
// Both sandbox.Create() and sandbox.Host() return a unified Computer object.
//
// # JavaScript API
//
//	const pc   = sandbox.Create({ image: "node:20", owner: "user1" }) // → Computer (kind="box")
//	const pc   = sandbox.Get(id)               // → Computer (kind="box") | null
//	const list = sandbox.List({ owner: "u1" }) // → BoxInfo[]
//	sandbox.Delete(id)                          // → void
//	const host = sandbox.Host("gpu")            // → Computer (kind="host")
//	const node = sandbox.GetNode("tai-abc123") // → NodeInfo | null
//	const all  = sandbox.Nodes()               // → NodeInfo[]
//	const team = sandbox.NodesByTeam("t-001")  // → NodeInfo[]
//
// # Go mapping
//
//	sandbox.Create(opts)  → Manager.Create(ctx, CreateOptions)    → Computer (Box)
//	sandbox.Create(opts)  → Manager.GetOrCreate(ctx, opts)        → Computer (Box)  (when opts.id is set)
//	sandbox.Get(id)       → Manager.Get(ctx, id)                  → Computer (Box)
//	sandbox.List(filter?) → Manager.List(ctx, ListOptions)        → BoxInfo[]
//	sandbox.Delete(id)    → Manager.Remove(ctx, id)               → void
//	sandbox.Host(nodeID?) → Manager.Host(ctx, nodeID)             → Computer (Host)
//	sandbox.GetNode(id)   → registry.Global().Get(id)             → NodeInfo | null
//	sandbox.Nodes()       → registry.Global().List()              → NodeInfo[]
//	sandbox.NodesByTeam(t)→ registry.Global().ListByTeam(t)       → NodeInfo[]
//
// Registration happens via init() — import with:
//
//	_ "github.com/yaoapp/yao/sandbox/v2/jsapi"
package jsapi

import (
	"context"
	"encoding/json"
	"time"

	v8 "github.com/yaoapp/gou/runtime/v8"
	sandbox "github.com/yaoapp/yao/sandbox/v2"
	"rogchap.com/v8go"
)

func init() {
	v8.RegisterObject("sandbox", ExportObject)
}

// ExportObject exports the sandbox namespace object to V8.
func ExportObject(iso *v8go.Isolate) *v8go.ObjectTemplate {
	obj := v8go.NewObjectTemplate(iso)
	obj.Set("Create", v8go.NewFunctionTemplate(iso, sbCreate))
	obj.Set("Get", v8go.NewFunctionTemplate(iso, sbGet))
	obj.Set("List", v8go.NewFunctionTemplate(iso, sbList))
	obj.Set("Delete", v8go.NewFunctionTemplate(iso, sbDelete))
	obj.Set("Host", v8go.NewFunctionTemplate(iso, sbHost))
	obj.Set("GetNode", v8go.NewFunctionTemplate(iso, sbGetNode))
	obj.Set("Nodes", v8go.NewFunctionTemplate(iso, sbNodes))
	obj.Set("NodesByTeam", v8go.NewFunctionTemplate(iso, sbNodesByTeam))
	return obj
}

// sbCreate: `sandbox.Create(options)` → Computer (kind="box")
func sbCreate(info *v8go.FunctionCallbackInfo) *v8go.Value {
	v8ctx := info.Context()
	ctx := context.Background()
	args := info.Args()
	if len(args) < 1 || !args[0].IsObject() {
		return throwError(info, "Create requires options object")
	}

	optsVal := args[0]
	jsonStr, err := v8go.JSONStringify(v8ctx, optsVal)
	if err != nil {
		return throwError(info, "Create: invalid options: "+err.Error())
	}

	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &raw); err != nil {
		return throwError(info, "Create: invalid options JSON: "+err.Error())
	}

	opts := sandbox.CreateOptions{}
	if v, ok := raw["id"].(string); ok {
		opts.ID = v
	}
	if v, ok := raw["owner"].(string); ok {
		opts.Owner = v
	}
	if v, ok := raw["node_id"].(string); ok {
		opts.NodeID = v
	}
	if v, ok := raw["image"].(string); ok {
		opts.Image = v
	}
	if v, ok := raw["workdir"].(string); ok {
		opts.WorkDir = v
	}
	if v, ok := raw["user"].(string); ok {
		opts.User = v
	}
	if v, ok := raw["env"].(map[string]interface{}); ok {
		env := make(map[string]string, len(v))
		for k, val := range v {
			if s, ok := val.(string); ok {
				env[k] = s
			}
		}
		opts.Env = env
	}
	if v, ok := raw["memory"].(float64); ok {
		opts.Memory = int64(v)
	}
	if v, ok := raw["cpus"].(float64); ok {
		opts.CPUs = v
	}
	if v, ok := raw["vnc"].(bool); ok {
		opts.VNC = v
	}
	if v, ok := raw["policy"].(string); ok {
		opts.Policy = sandbox.LifecyclePolicy(v)
	}
	if v, ok := raw["idle_timeout"].(float64); ok {
		opts.IdleTimeout = time.Duration(v) * time.Millisecond
	}
	if v, ok := raw["stop_timeout"].(float64); ok {
		opts.StopTimeout = time.Duration(v) * time.Millisecond
	}
	if v, ok := raw["workspace_id"].(string); ok {
		opts.WorkspaceID = v
	}
	if v, ok := raw["mount_mode"].(string); ok {
		opts.MountMode = v
	}
	if v, ok := raw["mount_path"].(string); ok {
		opts.MountPath = v
	}
	if v, ok := raw["labels"].(map[string]interface{}); ok {
		labels := make(map[string]string, len(v))
		for k, val := range v {
			if s, ok := val.(string); ok {
				labels[k] = s
			}
		}
		opts.Labels = labels
	}
	if v, ok := raw["ports"].([]interface{}); ok {
		for _, p := range v {
			pm, ok := p.(map[string]interface{})
			if !ok {
				continue
			}
			mapping := sandbox.PortMapping{}
			if cp, ok := pm["container_port"].(float64); ok {
				mapping.ContainerPort = int(cp)
			}
			if hp, ok := pm["host_port"].(float64); ok {
				mapping.HostPort = int(hp)
			}
			if hi, ok := pm["host_ip"].(string); ok {
				mapping.HostIP = hi
			}
			if pr, ok := pm["protocol"].(string); ok {
				mapping.Protocol = pr
			}
			opts.Ports = append(opts.Ports, mapping)
		}
	}

	m := sandbox.M()
	var box *sandbox.Box
	if opts.ID != "" {
		box, err = m.GetOrCreate(ctx, opts)
	} else {
		box, err = m.Create(ctx, opts)
	}
	if err != nil {
		return throwError(info, err.Error())
	}

	val, err := NewComputerObject(v8ctx, "box", box.ID())
	if err != nil {
		return throwError(info, err.Error())
	}
	return val
}

// sbGet: `sandbox.Get(id)` → Computer (kind="box") | null
func sbGet(info *v8go.FunctionCallbackInfo) *v8go.Value {
	iso := info.Context().Isolate()
	v8ctx := info.Context()
	ctx := context.Background()
	args := info.Args()
	if len(args) < 1 || !args[0].IsString() {
		return throwError(info, "Get requires id (string)")
	}
	id := args[0].String()

	_, err := sandbox.M().Get(ctx, id)
	if err != nil {
		return v8go.Null(iso)
	}

	val, err := NewComputerObject(v8ctx, "box", id)
	if err != nil {
		return throwError(info, err.Error())
	}
	return val
}

// sbList: `sandbox.List(filter?)` → BoxInfo[]
func sbList(info *v8go.FunctionCallbackInfo) *v8go.Value {
	v8ctx := info.Context()
	ctx := context.Background()
	args := info.Args()

	opts := sandbox.ListOptions{}
	if len(args) > 0 && args[0].IsObject() {
		jsonStr, _ := v8go.JSONStringify(v8ctx, args[0])
		var raw map[string]interface{}
		if json.Unmarshal([]byte(jsonStr), &raw) == nil {
			if v, ok := raw["owner"].(string); ok {
				opts.Owner = v
			}
			if v, ok := raw["node_id"].(string); ok {
				opts.NodeID = v
			}
			if v, ok := raw["labels"].(map[string]interface{}); ok {
				labels := make(map[string]string, len(v))
				for k, val := range v {
					if s, ok := val.(string); ok {
						labels[k] = s
					}
				}
				opts.Labels = labels
			}
		}
	}

	boxes, err := sandbox.M().List(ctx, opts)
	if err != nil {
		return throwError(info, err.Error())
	}

	items := make([]interface{}, 0, len(boxes))
	for _, b := range boxes {
		bi, err := b.Info(ctx)
		if err != nil {
			continue
		}
		items = append(items, map[string]interface{}{
			"id":            bi.ID,
			"container_id":  bi.ContainerID,
			"node_id":       bi.NodeID,
			"owner":         bi.Owner,
			"status":        bi.Status,
			"image":         bi.Image,
			"vnc":           bi.VNC,
			"policy":        string(bi.Policy),
			"labels":        bi.Labels,
			"created_at":    bi.CreatedAt.Format(time.RFC3339),
			"last_active":   bi.LastActive.Format(time.RFC3339),
			"process_count": bi.ProcessCount,
		})
	}

	data, _ := json.Marshal(items)
	val, _ := v8go.JSONParse(v8ctx, string(data))
	return val
}

// sbDelete: `sandbox.Delete(id)` → void
func sbDelete(info *v8go.FunctionCallbackInfo) *v8go.Value {
	iso := info.Context().Isolate()
	args := info.Args()
	if len(args) < 1 || !args[0].IsString() {
		return throwError(info, "Delete requires id (string)")
	}
	ctx := context.Background()
	id := args[0].String()

	if err := sandbox.M().Remove(ctx, id); err != nil {
		return throwError(info, err.Error())
	}
	return v8go.Undefined(iso)
}
