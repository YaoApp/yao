package mobile

import (
	"context"
	_ "embed"
	"fmt"
	"strings"

	"github.com/yaoapp/gou/process"
)

//go:embed screenshot_schema.json
var ScreenshotSchemaJSON []byte

const screenshotPath = "/sdcard/tai_screen.png"

// ScreenshotHandler is the tools.mobile_screenshot process handler.
// Takes a screenshot on the Android device and returns it as a base64-encoded PNG.
//
// Args[0]: device_id (string)
func ScreenshotHandler(proc *process.Process) interface{} {
	deviceID, err := extractDeviceID(proc, 0)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}

	if _, err := getAndroidNode(deviceID); err != nil {
		return map[string]any{"error": err.Error()}
	}

	ctx := context.Background()
	if proc.Context != nil {
		ctx = proc.Context
	}

	captureCmd := fmt.Sprintf("screencap -p %s", screenshotPath)
	_, stderr, exitCode, err := execOnDevice(ctx, deviceID, captureCmd)
	if err != nil || exitCode != 0 {
		return map[string]any{
			"error": fmt.Sprintf("screencap failed (exit=%d): %s %v", exitCode, strings.TrimSpace(stderr), err),
		}
	}

	encodeCmd := fmt.Sprintf("base64 %s | tr -d '\\n'", screenshotPath)
	stdout, stderr, exitCode, err := execOnDevice(ctx, deviceID, encodeCmd)
	if err != nil || exitCode != 0 {
		return map[string]any{
			"error": fmt.Sprintf("base64 encode failed (exit=%d): %s %v", exitCode, strings.TrimSpace(stderr), err),
		}
	}

	// Cleanup temp file
	go func() {
		execOnDevice(context.Background(), deviceID, "rm -f "+screenshotPath)
	}()

	return map[string]any{
		"image_base64": strings.TrimSpace(stdout),
		"format":       "png",
	}
}
