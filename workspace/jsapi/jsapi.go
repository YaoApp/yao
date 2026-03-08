// Package jsapi registers the workspace namespace into the Yao V8 runtime.
package jsapi

import (
	"context"
	"encoding/json"

	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/yao/workspace"
	"rogchap.com/v8go"
)

func init() {
	v8.RegisterObject("workspace", ExportObject)
}

func ExportObject(iso *v8go.Isolate) *v8go.ObjectTemplate {
	obj := v8go.NewObjectTemplate(iso)
	obj.Set("Create", v8go.NewFunctionTemplate(iso, wsCreate))
	obj.Set("Get", v8go.NewFunctionTemplate(iso, wsGet))
	obj.Set("List", v8go.NewFunctionTemplate(iso, wsList))
	obj.Set("Delete", v8go.NewFunctionTemplate(iso, wsDelete))
	return obj
}

func wsCreate(info *v8go.FunctionCallbackInfo) *v8go.Value {
	ctx := context.Background()

	args := info.Args()
	if len(args) < 1 || !args[0].IsObject() {
		return throwError(info, "workspace.Create requires an options object")
	}

	optsObj, err := args[0].AsObject()
	if err != nil {
		return throwError(info, "invalid options: "+err.Error())
	}

	opts := workspace.CreateOptions{}
	if v, e := optsObj.Get("id"); e == nil && v.IsString() {
		opts.ID = v.String()
	}
	if v, e := optsObj.Get("name"); e == nil && v.IsString() {
		opts.Name = v.String()
	}
	if v, e := optsObj.Get("owner"); e == nil && v.IsString() {
		opts.Owner = v.String()
	}
	if v, e := optsObj.Get("node"); e == nil && v.IsString() {
		opts.Node = v.String()
	}
	if v, e := optsObj.Get("labels"); e == nil && v.IsObject() {
		opts.Labels = parseStringMapFromValue(info.Context(), v)
	}

	if opts.Name == "" || opts.Owner == "" || opts.Node == "" {
		return throwError(info, "workspace.Create: name, owner, and node are required")
	}

	ws, err := workspace.M().Create(ctx, opts)
	if err != nil {
		return throwError(info, err.Error())
	}

	val, err := NewFSObject(info.Context(), ws.ID)
	if err != nil {
		return throwError(info, err.Error())
	}
	return val
}

func wsGet(info *v8go.FunctionCallbackInfo) *v8go.Value {
	iso := info.Context().Isolate()
	ctx := context.Background()

	args := info.Args()
	if len(args) < 1 || !args[0].IsString() {
		return throwError(info, "workspace.Get requires a string ID")
	}

	id := args[0].String()
	_, err := workspace.M().Get(ctx, id)
	if err != nil {
		return v8go.Null(iso)
	}

	val, err := NewFSObject(info.Context(), id)
	if err != nil {
		return v8go.Null(iso)
	}
	return val
}

func wsList(info *v8go.FunctionCallbackInfo) *v8go.Value {
	ctx := context.Background()
	v8ctx := info.Context()

	opts := workspace.ListOptions{}
	args := info.Args()
	if len(args) > 0 && args[0].IsObject() {
		filterObj, _ := args[0].AsObject()
		if filterObj != nil {
			if v, e := filterObj.Get("owner"); e == nil && v.IsString() {
				opts.Owner = v.String()
			}
			if v, e := filterObj.Get("node"); e == nil && v.IsString() {
				opts.Node = v.String()
			}
		}
	}

	list, err := workspace.M().List(ctx, opts)
	if err != nil {
		return throwError(info, err.Error())
	}

	type wsInfo struct {
		ID        string            `json:"id"`
		Name      string            `json:"name"`
		Owner     string            `json:"owner"`
		Node      string            `json:"node"`
		Labels    map[string]string `json:"labels,omitempty"`
		CreatedAt string            `json:"created_at"`
		UpdatedAt string            `json:"updated_at"`
	}

	items := make([]wsInfo, len(list))
	for i, ws := range list {
		items[i] = wsInfo{
			ID: ws.ID, Name: ws.Name, Owner: ws.Owner, Node: ws.Node,
			Labels: ws.Labels, CreatedAt: ws.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt: ws.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	data, _ := json.Marshal(items)
	val, err := v8go.JSONParse(v8ctx, string(data))
	if err != nil {
		return throwError(info, err.Error())
	}
	return val
}

func wsDelete(info *v8go.FunctionCallbackInfo) *v8go.Value {
	iso := info.Context().Isolate()
	ctx := context.Background()

	args := info.Args()
	if len(args) < 1 || !args[0].IsString() {
		return throwError(info, "workspace.Delete requires a string ID")
	}

	if err := workspace.M().Delete(ctx, args[0].String(), false); err != nil {
		return throwError(info, err.Error())
	}
	return v8go.Undefined(iso)
}

func throwError(info *v8go.FunctionCallbackInfo, msg string) *v8go.Value {
	iso := info.Context().Isolate()
	e, _ := v8go.NewValue(iso, msg)
	iso.ThrowException(e)
	return v8go.Undefined(iso)
}

func parseStringMapFromValue(v8ctx *v8go.Context, val *v8go.Value) map[string]string {
	result := make(map[string]string)
	jsonStr, err := v8go.JSONStringify(v8ctx, val)
	if err != nil {
		return result
	}
	_ = json.Unmarshal([]byte(jsonStr), &result)
	return result
}
