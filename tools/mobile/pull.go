package mobile

import (
	"context"
	_ "embed"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/yaoapp/gou/process"
	tai "github.com/yaoapp/yao/tai"
)

//go:embed pull_schema.json
var PullSchemaJSON []byte

// PullHandler is the tools.mobile_pull process handler.
// Pulls a file from the Android device to the Yao server (or returns base64).
//
// Args[0]: device_id (string)
// Args[1]: remote_path (string) - source path on the device
// Args[2]: local_path (string, optional) - destination path on the Yao server; if empty, returns content as base64
func PullHandler(proc *process.Process) interface{} {
	deviceID, err := extractDeviceID(proc, 0)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}

	remotePath := proc.ArgsString(1)
	localPath := proc.ArgsString(2)

	if remotePath == "" {
		return map[string]any{"error": "remote_path is required"}
	}

	if _, err := getAndroidNode(deviceID); err != nil {
		return map[string]any{"error": err.Error()}
	}

	res, ok := tai.GetResources(deviceID)
	if !ok {
		return map[string]any{"error": fmt.Sprintf("device %q: no connection resources", deviceID)}
	}
	if res.Volume == nil {
		return map[string]any{"error": fmt.Sprintf("device %q: volume not enabled", deviceID)}
	}

	ctx := context.Background()
	if proc.Context != nil {
		ctx = proc.Context
	}

	data, _, err := res.Volume.ReadFile(ctx, deviceID, remotePath)
	if err != nil {
		return map[string]any{"error": fmt.Sprintf("read from device: %v", err)}
	}

	if localPath != "" {
		if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
			return map[string]any{"error": fmt.Sprintf("create local dir: %v", err)}
		}
		if err := os.WriteFile(localPath, data, 0644); err != nil {
			return map[string]any{"error": fmt.Sprintf("write local file: %v", err)}
		}
		return map[string]any{
			"success":    true,
			"local_path": localPath,
			"bytes":      len(data),
		}
	}

	return map[string]any{
		"content_base64": base64.StdEncoding.EncodeToString(data),
		"remote_path":    remotePath,
		"bytes":          len(data),
	}
}
