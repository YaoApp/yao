package sandboxv2

import (
	"context"
	"fmt"
	"log"
	"path"
	"strings"

	"github.com/yaoapp/yao/agent/sandbox/v2/types"
	infra "github.com/yaoapp/yao/sandbox/v2"
	"github.com/yaoapp/yao/tai/workspace"
)

const onceMarkerDir = ".yao/prepare"

// RunPrepareSteps executes a list of PrepareStep actions on the given Computer.
// file/copy/marker operations use computer.Workplace() (gRPC volume, cross-platform).
// exec operations use shell via Computer.Exec.
func RunPrepareSteps(ctx context.Context, steps []types.PrepareStep, computer infra.Computer, assistantID, configHash string) error {
	if len(steps) == 0 {
		return nil
	}

	var ws workspace.FS
	if computer != nil {
		ws = computer.Workplace()
	}

	markerDir := onceMarkerDir
	if assistantID != "" {
		markerDir = onceMarkerDir + "/" + assistantID
	}
	markerPath := markerDir + "/done"

	skipOnce := false
	if configHash != "" && ws != nil {
		if data, err := ws.ReadFile(markerPath); err == nil {
			if strings.TrimSpace(string(data)) == configHash {
				skipOnce = true
			}
		}
	}

	for i, step := range steps {
		if step.Once && skipOnce {
			continue
		}

		var err error
		switch step.Action {
		case "file":
			err = runFileStep(ws, step)
		case "copy":
			err = runCopyStep(ws, step)
		case "exec":
			err = runExecStep(ctx, computer, step)
		case "process":
			log.Printf("[sandbox/v2] prepare step %d: action=process (reserved, skipping)", i)
		default:
			err = fmt.Errorf("unknown prepare action %q", step.Action)
		}

		if err != nil {
			if step.IgnoreError {
				log.Printf("[sandbox/v2] prepare step %d (%s): ignored error: %v", i, step.Action, err)
				continue
			}
			return fmt.Errorf("prepare step %d (%s): %w", i, step.Action, err)
		}
	}

	if configHash != "" && ws != nil {
		ws.MkdirAll(markerDir, 0755)
		ws.WriteFile(markerPath, []byte(configHash), 0644)
	}

	return nil
}

// ---------------------------------------------------------------------------
// Step runners
// ---------------------------------------------------------------------------

func runFileStep(ws workspace.FS, step types.PrepareStep) error {
	if step.Path == "" {
		return fmt.Errorf("file step requires path")
	}
	if ws == nil {
		return fmt.Errorf("file step requires workspace")
	}

	dir := path.Dir(step.Path)
	if dir != "." && dir != "/" {
		if err := ws.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}

	if err := ws.WriteFile(step.Path, step.Content, 0644); err != nil {
		return fmt.Errorf("write file %s: %w", step.Path, err)
	}
	return nil
}

func runCopyStep(ws workspace.FS, step types.PrepareStep) error {
	if step.Src == "" || step.Dst == "" {
		return fmt.Errorf("copy step requires src and dst")
	}
	if ws == nil {
		return fmt.Errorf("copy step requires workspace")
	}

	data, err := ws.ReadFile(step.Src)
	if err != nil {
		return fmt.Errorf("read src %s: %w", step.Src, err)
	}

	dir := path.Dir(step.Dst)
	if dir != "." && dir != "/" {
		if err := ws.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}

	if err := ws.WriteFile(step.Dst, data, 0644); err != nil {
		return fmt.Errorf("write dst %s: %w", step.Dst, err)
	}
	return nil
}

func runExecStep(ctx context.Context, computer infra.Computer, step types.PrepareStep) error {
	if step.Cmd == "" {
		return fmt.Errorf("exec step requires cmd")
	}

	kind := shellFromSystem(computer)
	script := step.Cmd
	if step.Background {
		if kind == shellSh {
			script = fmt.Sprintf("nohup %s > /dev/null 2>&1 &", step.Cmd)
		} else {
			script = fmt.Sprintf("Start-Process -NoNewWindow -FilePath 'cmd.exe' -ArgumentList '/C %s'", step.Cmd)
		}
	}

	result, err := computer.Exec(ctx, shellWrap(kind, script), infra.WithWorkDir("/"))
	if err != nil {
		return err
	}
	label := "exec"
	if step.Background {
		label = "exec(background)"
	}
	return checkResult(result, label)
}

// checkResult inspects ExecResult for errors.
func checkResult(result *infra.ExecResult, label string) error {
	if result.Error != "" {
		return fmt.Errorf("%s: %s", label, result.Error)
	}
	if result.ExitCode != 0 {
		stderr := result.Stderr
		if len(stderr) > 200 {
			stderr = stderr[:200] + "..."
		}
		return fmt.Errorf("%s: exit %d: %s", label, result.ExitCode, stderr)
	}
	return nil
}
