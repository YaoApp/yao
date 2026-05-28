package workspace

import (
	_ "embed"
	"encoding/base64"

	"github.com/yaoapp/gou/process"
	ws "github.com/yaoapp/yao/workspace"
)

//go:embed file_list_schema.json
var FileListSchemaJSON []byte

//go:embed file_read_schema.json
var FileReadSchemaJSON []byte

//go:embed file_write_schema.json
var FileWriteSchemaJSON []byte

// FileListHandler is the tools.workspace_file_list process handler.
//
// Args[0]: id (string, required) — workspace ID
// Args[1]: path (string, optional) — directory path, default "."
func FileListHandler(proc *process.Process) interface{} {
	auth := proc.Authorized
	if auth == nil {
		return map[string]any{"error": "unauthorized"}
	}

	id := proc.ArgsString(0)
	if id == "" {
		return map[string]any{"error": "id is required"}
	}

	if _, err := resolveAndCheck(proc, id); err != nil {
		return map[string]any{"error": err.Error()}
	}

	path := proc.ArgsString(1)
	if path == "" {
		path = "."
	}

	m := ws.M()
	entries, err := m.ListDir(proc.Context, id, path)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}

	result := make([]map[string]any, 0, len(entries))
	for _, e := range entries {
		entry := map[string]any{
			"name":   e.Name,
			"is_dir": e.IsDir,
			"size":   e.Size,
		}
		if !e.ModTime.IsZero() {
			entry["mod_time"] = e.ModTime.Format("2006-01-02T15:04:05Z")
		}
		result = append(result, entry)
	}
	return map[string]any{"entries": result}
}

// FileReadHandler is the tools.workspace_file_read process handler.
//
// Args[0]: id (string, required) — workspace ID
// Args[1]: path (string, required) — file path
// Args[2]: encoding (string, optional) — "text" (default) or "base64"
func FileReadHandler(proc *process.Process) interface{} {
	auth := proc.Authorized
	if auth == nil {
		return map[string]any{"error": "unauthorized"}
	}

	id := proc.ArgsString(0)
	if id == "" {
		return map[string]any{"error": "id is required"}
	}

	path := proc.ArgsString(1)
	if path == "" {
		return map[string]any{"error": "path is required"}
	}

	if _, err := resolveAndCheck(proc, id); err != nil {
		return map[string]any{"error": err.Error()}
	}

	m := ws.M()
	data, err := m.ReadFile(proc.Context, id, path)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}

	encoding := proc.ArgsString(2)
	if encoding == "base64" {
		return map[string]any{
			"content":  base64.StdEncoding.EncodeToString(data),
			"encoding": "base64",
		}
	}

	return map[string]any{
		"content":  string(data),
		"encoding": "text",
	}
}

// FileWriteHandler is the tools.workspace_file_write process handler.
//
// Args[0]: id (string, required) — workspace ID
// Args[1]: path (string, required) — file path
// Args[2]: content (string, required) — file content
// Args[3]: encoding (string, optional) — "text" (default) or "base64"
func FileWriteHandler(proc *process.Process) interface{} {
	auth := proc.Authorized
	if auth == nil {
		return map[string]any{"error": "unauthorized"}
	}

	id := proc.ArgsString(0)
	if id == "" {
		return map[string]any{"error": "id is required"}
	}

	path := proc.ArgsString(1)
	if path == "" {
		return map[string]any{"error": "path is required"}
	}

	content := proc.ArgsString(2)

	if _, err := resolveAndCheck(proc, id); err != nil {
		return map[string]any{"error": err.Error()}
	}

	var data []byte
	encoding := proc.ArgsString(3)
	if encoding == "base64" {
		var err error
		data, err = base64.StdEncoding.DecodeString(content)
		if err != nil {
			return map[string]any{"error": "invalid base64 content: " + err.Error()}
		}
	} else {
		data = []byte(content)
	}

	m := ws.M()
	if err := m.WriteFile(proc.Context, id, path, data, 0644); err != nil {
		return map[string]any{"error": err.Error()}
	}

	return map[string]any{"message": "ok"}
}
