package mobile

import (
	"context"
	_ "embed"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/yaoapp/gou/process"
	tai "github.com/yaoapp/yao/tai"
)

//go:embed push_schema.json
var PushSchemaJSON []byte

// PushHandler is the tools.mobile_push process handler.
// Pushes a file from the Yao server (or base64 content) to the Android device.
//
// Args[0]: device_id (string)
// Args[1]: local_path (string) - path on the Yao server, OR empty if content is provided
// Args[2]: remote_path (string) - destination path on the device
// Args[3]: content_base64 (string, optional) - base64-encoded content to push directly
func PushHandler(proc *process.Process) interface{} {
	deviceID, err := extractDeviceID(proc, 0)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}

	localPath := proc.ArgsString(1)
	remotePath := proc.ArgsString(2)
	contentB64 := proc.ArgsString(3)

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

	var data []byte
	if contentB64 != "" {
		data, err = base64.StdEncoding.DecodeString(contentB64)
		if err != nil {
			return map[string]any{"error": fmt.Sprintf("invalid base64 content: %v", err)}
		}
	} else if localPath != "" {
		data, err = os.ReadFile(localPath)
		if err != nil {
			return map[string]any{"error": fmt.Sprintf("read local file: %v", err)}
		}
	} else {
		return map[string]any{"error": "either local_path or content_base64 is required"}
	}

	ctx := context.Background()
	if proc.Context != nil {
		ctx = proc.Context
	}

	if err := res.Volume.WriteFile(ctx, deviceID, remotePath, data, 0644); err != nil {
		return map[string]any{"error": fmt.Sprintf("write to device: %v", err)}
	}

	return map[string]any{
		"success":     true,
		"remote_path": remotePath,
		"bytes":       len(data),
	}
}
