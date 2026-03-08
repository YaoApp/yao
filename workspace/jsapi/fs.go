package jsapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/tai/volume"
	"github.com/yaoapp/yao/workspace"
	"rogchap.com/v8go"
)

// NewFSObject creates a JS WorkspaceFS object backed by a workspace ID string.
// All methods delegate to workspace.M() — no Go object is passed to V8.
func NewFSObject(v8ctx *v8go.Context, workspaceID string) (*v8go.Value, error) {
	iso := v8ctx.Isolate()
	ctx := context.Background()
	wsID := workspaceID

	ws, err := workspace.M().Get(ctx, wsID)
	if err != nil {
		return nil, fmt.Errorf("workspace %s: %w", wsID, err)
	}

	tpl := v8go.NewObjectTemplate(iso)

	tpl.Set("ReadFile", v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			return throwError(info, "ReadFile requires a path")
		}
		data, err := workspace.M().ReadFile(ctx, wsID, args[0].String())
		if err != nil {
			return throwError(info, err.Error())
		}
		val, _ := v8go.NewValue(iso, string(data))
		return val
	}))

	tpl.Set("WriteFile", v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 2 {
			return throwError(info, "WriteFile requires path and data")
		}
		perm := os.FileMode(0o644)
		if len(args) > 2 && args[2].IsNumber() {
			perm = os.FileMode(args[2].Int32())
		}
		if err := workspace.M().WriteFile(ctx, wsID, args[0].String(), []byte(args[1].String()), perm); err != nil {
			return throwError(info, err.Error())
		}
		return v8go.Undefined(iso)
	}))

	tpl.Set("ReadDir", v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		path := "."
		args := info.Args()
		if len(args) > 0 && args[0].IsString() {
			path = args[0].String()
		}
		recursive := false
		if len(args) > 1 && args[1].IsBoolean() {
			recursive = args[1].Boolean()
		}

		if recursive {
			return readDirRecursive(info, wsID, path)
		}

		entries, err := workspace.M().ListDir(ctx, wsID, path)
		if err != nil {
			return throwError(info, err.Error())
		}
		data, _ := json.Marshal(entries)
		val, _ := v8go.JSONParse(info.Context(), string(data))
		return val
	}))

	tpl.Set("Stat", v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			return throwError(info, "Stat requires a path")
		}
		fsys, err := workspace.M().FS(ctx, wsID)
		if err != nil {
			return throwError(info, err.Error())
		}
		fi, err := fsys.Stat(args[0].String())
		if err != nil {
			return throwError(info, err.Error())
		}
		return fileInfoToJS(info, fi)
	}))

	tpl.Set("MkdirAll", v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			return throwError(info, "MkdirAll requires a path")
		}
		if err := workspace.M().MkdirAll(ctx, wsID, args[0].String()); err != nil {
			return throwError(info, err.Error())
		}
		return v8go.Undefined(iso)
	}))

	tpl.Set("Remove", v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			return throwError(info, "Remove requires a path")
		}
		fsys, err := workspace.M().FS(ctx, wsID)
		if err != nil {
			return throwError(info, err.Error())
		}
		if err := fsys.Remove(args[0].String()); err != nil {
			return throwError(info, err.Error())
		}
		return v8go.Undefined(iso)
	}))

	tpl.Set("RemoveAll", v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			return throwError(info, "RemoveAll requires a path")
		}
		fsys, err := workspace.M().FS(ctx, wsID)
		if err != nil {
			return throwError(info, err.Error())
		}
		if err := fsys.RemoveAll(args[0].String()); err != nil {
			return throwError(info, err.Error())
		}
		return v8go.Undefined(iso)
	}))

	tpl.Set("Rename", v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 2 {
			return throwError(info, "Rename requires from and to paths")
		}
		if err := workspace.M().Rename(ctx, wsID, args[0].String(), args[1].String()); err != nil {
			return throwError(info, err.Error())
		}
		return v8go.Undefined(iso)
	}))

	tpl.Set("ReadFileBase64", v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			return throwError(info, "ReadFileBase64 requires a path")
		}
		data, err := workspace.M().ReadFile(ctx, wsID, args[0].String())
		if err != nil {
			return throwError(info, err.Error())
		}
		val, _ := v8go.NewValue(iso, base64.StdEncoding.EncodeToString(data))
		return val
	}))

	tpl.Set("WriteFileBase64", v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 2 {
			return throwError(info, "WriteFileBase64 requires path and b64 data")
		}
		perm := os.FileMode(0o644)
		if len(args) > 2 && args[2].IsNumber() {
			perm = os.FileMode(args[2].Int32())
		}
		decoded, err := base64.StdEncoding.DecodeString(args[1].String())
		if err != nil {
			return throwError(info, "base64 decode: "+err.Error())
		}
		if err := workspace.M().WriteFile(ctx, wsID, args[0].String(), decoded, perm); err != nil {
			return throwError(info, err.Error())
		}
		return v8go.Undefined(iso)
	}))

	tpl.Set("ReadFileBuffer", v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			return throwError(info, "ReadFileBuffer requires a path")
		}
		data, err := workspace.M().ReadFile(ctx, wsID, args[0].String())
		if err != nil {
			return throwError(info, err.Error())
		}
		val, _ := v8go.NewValue(iso, base64.StdEncoding.EncodeToString(data))
		return val
	}))

	tpl.Set("WriteFileBuffer", v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 2 {
			return throwError(info, "WriteFileBuffer requires path and data")
		}
		perm := os.FileMode(0o644)
		if len(args) > 2 && args[2].IsNumber() {
			perm = os.FileMode(args[2].Int32())
		}
		decoded, err := base64.StdEncoding.DecodeString(args[1].String())
		if err != nil {
			return throwError(info, "decode: "+err.Error())
		}
		if err := workspace.M().WriteFile(ctx, wsID, args[0].String(), decoded, perm); err != nil {
			return throwError(info, err.Error())
		}
		return v8go.Undefined(iso)
	}))

	tpl.Set("Exists", v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			val, _ := v8go.NewValue(iso, false)
			return val
		}
		fsys, err := workspace.M().FS(ctx, wsID)
		if err != nil {
			val, _ := v8go.NewValue(iso, false)
			return val
		}
		_, err = fsys.Stat(args[0].String())
		val, _ := v8go.NewValue(iso, err == nil)
		return val
	}))

	tpl.Set("IsDir", v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			val, _ := v8go.NewValue(iso, false)
			return val
		}
		fsys, err := workspace.M().FS(ctx, wsID)
		if err != nil {
			val, _ := v8go.NewValue(iso, false)
			return val
		}
		fi, err := fsys.Stat(args[0].String())
		val, _ := v8go.NewValue(iso, err == nil && fi.IsDir())
		return val
	}))

	tpl.Set("IsFile", v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			val, _ := v8go.NewValue(iso, false)
			return val
		}
		fsys, err := workspace.M().FS(ctx, wsID)
		if err != nil {
			val, _ := v8go.NewValue(iso, false)
			return val
		}
		fi, err := fsys.Stat(args[0].String())
		val, _ := v8go.NewValue(iso, err == nil && !fi.IsDir())
		return val
	}))

	tpl.Set("Copy", v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		return copyHandler(info, wsID)
	}))

	tpl.Set("Zip", v8go.NewFunctionTemplate(iso, archiveHandler(iso, wsID, "zip")))
	tpl.Set("Unzip", v8go.NewFunctionTemplate(iso, archiveHandler(iso, wsID, "unzip")))
	tpl.Set("Gzip", v8go.NewFunctionTemplate(iso, archiveHandler(iso, wsID, "gzip")))
	tpl.Set("Gunzip", v8go.NewFunctionTemplate(iso, archiveHandler(iso, wsID, "gunzip")))
	tpl.Set("Tar", v8go.NewFunctionTemplate(iso, archiveHandler(iso, wsID, "tar")))
	tpl.Set("Untar", v8go.NewFunctionTemplate(iso, archiveHandler(iso, wsID, "untar")))
	tpl.Set("Tgz", v8go.NewFunctionTemplate(iso, archiveHandler(iso, wsID, "tgz")))
	tpl.Set("Untgz", v8go.NewFunctionTemplate(iso, archiveHandler(iso, wsID, "untgz")))

	obj, err := tpl.NewInstance(v8ctx)
	if err != nil {
		return nil, err
	}

	idVal, _ := v8go.NewValue(iso, ws.ID)
	obj.Set("id", idVal)
	nameVal, _ := v8go.NewValue(iso, ws.Name)
	obj.Set("name", nameVal)
	nodeVal, _ := v8go.NewValue(iso, ws.Node)
	obj.Set("node", nodeVal)

	return obj.Value, nil
}

func archiveHandler(iso *v8go.Isolate, wsID, op string) v8go.FunctionCallback {
	return func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		ctx := context.Background()
		args := info.Args()
		if len(args) < 2 {
			return throwError(info, op+": requires src and dst paths")
		}
		src := args[0].String()
		dst := args[1].String()

		var excludes []string
		if len(args) > 2 && args[2].IsObject() {
			excObj, _ := args[2].AsObject()
			if excObj != nil {
				if v, e := excObj.Get("excludes"); e == nil && v.IsObject() {
					excludes = parseStringArray(v)
				}
			}
		}

		vol, sid, err := workspace.M().Volume(ctx, wsID)
		if err != nil {
			return throwError(info, err.Error())
		}

		var result *volume.ArchiveResult
		switch op {
		case "zip":
			result, err = vol.Zip(ctx, sid, src, dst, excludes)
		case "unzip":
			result, err = vol.Unzip(ctx, sid, src, dst)
		case "gzip":
			result, err = vol.Gzip(ctx, sid, src, dst)
		case "gunzip":
			result, err = vol.Gunzip(ctx, sid, src, dst)
		case "tar":
			result, err = vol.Tar(ctx, sid, src, dst, excludes)
		case "untar":
			result, err = vol.Untar(ctx, sid, src, dst)
		case "tgz":
			result, err = vol.Tgz(ctx, sid, src, dst, excludes)
		case "untgz":
			result, err = vol.Untgz(ctx, sid, src, dst)
		}
		if err != nil {
			return throwError(info, err.Error())
		}

		data, _ := json.Marshal(map[string]interface{}{
			"size_bytes":  result.SizeBytes,
			"files_count": result.FilesCount,
		})
		val, _ := v8go.JSONParse(info.Context(), string(data))
		return val
	}
}

func readDirRecursive(info *v8go.FunctionCallbackInfo, wsID, path string) *v8go.Value {
	ctx := context.Background()
	fsys, err := workspace.M().FS(ctx, wsID)
	if err != nil {
		return throwError(info, err.Error())
	}

	type entry struct {
		Name  string `json:"name"`
		IsDir bool   `json:"is_dir"`
		Size  int64  `json:"size"`
	}
	var entries []entry

	_ = fs.WalkDir(fsys, path, func(p string, d fs.DirEntry, err error) error {
		if err != nil || p == path {
			return err
		}
		rel, _ := filepath.Rel(path, p)
		var size int64
		if fi, e := d.Info(); e == nil {
			size = fi.Size()
		}
		entries = append(entries, entry{Name: filepath.ToSlash(rel), IsDir: d.IsDir(), Size: size})
		return nil
	})

	data, _ := json.Marshal(entries)
	val, _ := v8go.JSONParse(info.Context(), string(data))
	return val
}

func fileInfoToJS(info *v8go.FunctionCallbackInfo, fi fs.FileInfo) *v8go.Value {
	data, _ := json.Marshal(map[string]interface{}{
		"name":     fi.Name(),
		"size":     fi.Size(),
		"is_dir":   fi.IsDir(),
		"mod_time": fi.ModTime().Format(time.RFC3339),
		"mode":     uint32(fi.Mode()),
	})
	val, _ := v8go.JSONParse(info.Context(), string(data))
	return val
}

func copyHandler(info *v8go.FunctionCallbackInfo, wsID string) *v8go.Value {
	iso := info.Context().Isolate()
	ctx := context.Background()
	args := info.Args()
	if len(args) < 2 {
		return throwError(info, "Copy requires src and dst paths")
	}

	src := args[0].String()
	dst := args[1].String()
	srcIsHost := isHostURI(src)
	dstIsHost := isHostURI(dst)

	var excludes []string
	force := false
	if len(args) > 2 && args[2].IsObject() {
		optsObj, _ := args[2].AsObject()
		if optsObj != nil {
			if v, e := optsObj.Get("excludes"); e == nil && v.IsObject() {
				excludes = parseStringArray(v)
			}
			if v, e := optsObj.Get("force"); e == nil && v.IsBoolean() {
				force = v.Boolean()
			}
		}
	}

	vol, sid, err := workspace.M().Volume(ctx, wsID)
	if err != nil {
		return throwError(info, err.Error())
	}

	opts := []volume.SyncOption{}
	if len(excludes) > 0 {
		opts = append(opts, volume.WithExcludes(excludes...))
	}
	if force {
		opts = append(opts, volume.WithForceFull())
	}

	switch {
	case !srcIsHost && !dstIsHost:
		if err := copyWithinWorkspace(ctx, wsID, src, dst); err != nil {
			return throwError(info, err.Error())
		}
		return v8go.Undefined(iso)

	case srcIsHost && !dstIsHost:
		hostPath, err := resolveHostPath(src)
		if err != nil {
			return throwError(info, err.Error())
		}
		opts = append(opts, volume.WithRemotePath(dst))
		result, err := vol.SyncPush(ctx, sid, hostPath, opts...)
		if err != nil {
			return throwError(info, err.Error())
		}
		return syncResultToJS(info, result)

	case !srcIsHost && dstIsHost:
		hostPath, err := resolveHostPath(dst)
		if err != nil {
			return throwError(info, err.Error())
		}
		opts = append(opts, volume.WithRemotePath(src))
		result, err := vol.SyncPull(ctx, sid, hostPath, opts...)
		if err != nil {
			return throwError(info, err.Error())
		}
		return syncResultToJS(info, result)

	case srcIsHost && dstIsHost:
		srcPath, err := resolveHostPath(src)
		if err != nil {
			return throwError(info, err.Error())
		}
		dstPath, err := resolveHostPath(dst)
		if err != nil {
			return throwError(info, err.Error())
		}
		if err := copyLocalToLocal(srcPath, dstPath, excludes); err != nil {
			return throwError(info, err.Error())
		}
		return v8go.Undefined(iso)
	}

	return v8go.Undefined(iso)
}

func syncResultToJS(info *v8go.FunctionCallbackInfo, r *volume.SyncResult) *v8go.Value {
	data, _ := json.Marshal(map[string]interface{}{
		"files_synced":      r.FilesSynced,
		"bytes_transferred": r.BytesTransferred,
		"duration_ms":       r.Duration.Milliseconds(),
	})
	val, _ := v8go.JSONParse(info.Context(), string(data))
	return val
}

func copyWithinWorkspace(ctx context.Context, wsID, src, dst string) error {
	fsys, err := workspace.M().FS(ctx, wsID)
	if err != nil {
		return err
	}

	fi, err := fsys.Stat(src)
	if err != nil {
		return err
	}

	if !fi.IsDir() {
		data, err := fsys.ReadFile(src)
		if err != nil {
			return err
		}
		return fsys.WriteFile(dst, data, fi.Mode())
	}

	return fs.WalkDir(fsys, src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return fsys.MkdirAll(target, 0o755)
		}
		data, err := fsys.ReadFile(p)
		if err != nil {
			return err
		}
		info, _ := d.Info()
		perm := os.FileMode(0o644)
		if info != nil {
			perm = info.Mode()
		}
		return fsys.WriteFile(target, data, perm)
	})
}

func isHostURI(path string) bool {
	return strings.HasPrefix(path, "local://") || strings.HasPrefix(path, "tmp://")
}

func resolveHostPath(rawPath string) (string, error) {
	if strings.HasPrefix(rawPath, "tmp://") {
		rel := strings.TrimPrefix(rawPath, "tmp://")
		if strings.Contains(rel, "..") {
			return "", fmt.Errorf("path traversal not allowed")
		}
		return filepath.Join(os.TempDir(), rel), nil
	}

	appRoot := config.Conf.AppSource
	rel := strings.TrimPrefix(rawPath, "local://")
	if strings.Contains(rel, "..") {
		return "", fmt.Errorf("path traversal not allowed")
	}
	abs := filepath.Join(appRoot, rel)
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		resolved = abs
	}
	if !strings.HasPrefix(resolved, appRoot) {
		return "", fmt.Errorf("path escapes app root")
	}
	return resolved, nil
}

func copyLocalToLocal(src, dst string, excludes []string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		data, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		return os.WriteFile(dst, data, info.Mode())
	}

	return filepath.WalkDir(src, func(abs string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, abs)
		if rel == "." {
			return os.MkdirAll(dst, 0o755)
		}
		for _, p := range excludes {
			if matched, _ := filepath.Match(p, filepath.Base(rel)); matched {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(abs)
		if err != nil {
			return err
		}
		fi, _ := d.Info()
		perm := os.FileMode(0o644)
		if fi != nil {
			perm = fi.Mode()
		}
		return os.WriteFile(target, data, perm)
	})
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
