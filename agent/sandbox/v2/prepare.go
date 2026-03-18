package sandboxv2

import (
	"context"
	"fmt"
	"log"
	"path"
	pathpkg "path/filepath"
	"strings"

	"github.com/yaoapp/yao/agent/sandbox/v2/types"
	infra "github.com/yaoapp/yao/sandbox/v2"
	"github.com/yaoapp/yao/tai/workspace"
)

const onceMarkerDir = ".yao/prepare"

// RunPrepareSteps executes a list of PrepareStep actions on the given Computer.
// file/copy/marker operations use computer.Workplace() (gRPC volume, cross-platform).
// exec operations use shell via Computer.Exec.
// assistantDir is the absolute host path to the assistant source directory;
// copy steps with a relative src resolve against it (host → workspace push).
func RunPrepareSteps(ctx context.Context, steps []types.PrepareStep, computer infra.Computer, assistantID, configHash, assistantDir string) error {
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
			err = runCopyStep(ws, step, assistantDir)
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

// runCopyStep copies files into the workspace using ws.Copy which supports
// the "local:///" URI scheme for host-to-workspace transfers.
//
// src resolution:
//   - Already a host URI ("local:///..." or "tmp:///...") → used as-is
//   - Relative path + assistantDir provided → resolved to "local:///<assistantDir>/<src>"
//   - Relative path without assistantDir → treated as workspace-internal path
func runCopyStep(ws workspace.FS, step types.PrepareStep, assistantDir string) error {
	if step.Src == "" || step.Dst == "" {
		return fmt.Errorf("copy step requires src and dst")
	}
	if ws == nil {
		return fmt.Errorf("copy step requires workspace")
	}

	src := step.Src
	if !isHostURI(src) && assistantDir != "" {
		src = "local:///" + pathpkg.Join(assistantDir, src)
	}

	if _, err := ws.Copy(src, step.Dst); err != nil {
		return fmt.Errorf("copy %s -> %s: %w", src, step.Dst, err)
	}
	return nil
}

func isHostURI(s string) bool {
	return strings.HasPrefix(s, "local:///") || strings.HasPrefix(s, "tmp:///")
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

	rootDir := "/"
	if isWindowsComputer(computer) {
		rootDir = `C:\`
	}

	result, err := computer.Exec(ctx, shellWrap(kind, script), infra.WithWorkDir(rootDir))
	if err != nil {
		return err
	}
	label := "exec"
	if step.Background {
		label = "exec(background)"
	}
	return checkResult(result, label)
}

func isWindowsComputer(computer infra.Computer) bool {
	return strings.EqualFold(computer.ComputerInfo().System.OS, "windows")
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
