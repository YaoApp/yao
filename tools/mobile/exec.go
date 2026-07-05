package mobile

import (
	"context"
	_ "embed"
	"strings"

	"github.com/yaoapp/gou/process"
)

//go:embed exec_schema.json
var ExecSchemaJSON []byte

// ExecHandler is the tools.mobile_exec process handler.
// Executes a shell command on the target Android device.
//
// Args[0]: device_id (string)
// Args[1]: command (string)
func ExecHandler(proc *process.Process) interface{} {
	deviceID, err := extractDeviceID(proc, 0)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}

	command := proc.ArgsString(1)
	if command == "" {
		return map[string]any{"error": "command is required"}
	}

	if _, err := getAndroidNode(deviceID); err != nil {
		return map[string]any{"error": err.Error()}
	}

	ctx := context.Background()
	if proc.Context != nil {
		ctx = proc.Context
	}

	stdout, stderr, exitCode, err := execOnDevice(ctx, deviceID, command)
	if err != nil {
		return map[string]any{
			"error":     err.Error(),
			"stdout":    strings.TrimSpace(stdout),
			"stderr":    strings.TrimSpace(stderr),
			"exit_code": exitCode,
		}
	}

	return map[string]any{
		"stdout":    strings.TrimSpace(stdout),
		"stderr":    strings.TrimSpace(stderr),
		"exit_code": exitCode,
	}
}
