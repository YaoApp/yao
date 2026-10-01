//go:build unit

package jobtool

import (
	"testing"

	"github.com/yaoapp/gou/process"
)

func TestJobStartHandler_ReturnsArgs(t *testing.T) {
	args := map[string]interface{}{
		"id":     "job_abc123",
		"kind":   "job",
		"status": "running",
	}
	proc := &process.Process{Args: []interface{}{args}}
	result := JobStartHandler(proc)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestDaemonStartHandler_ReturnsArgs(t *testing.T) {
	args := map[string]interface{}{
		"id":          "daemon_xyz",
		"kind":        "daemon",
		"status":      "running",
		"daemon_port": float64(3000),
	}
	proc := &process.Process{Args: []interface{}{args}}
	result := DaemonStartHandler(proc)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestJobSettledHandler_ReturnsArgs(t *testing.T) {
	args := map[string]interface{}{
		"id":     "job_settled1",
		"status": "completed",
	}
	proc := &process.Process{Args: []interface{}{args}}
	result := JobSettledHandler(proc)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestDaemonPortReadyHandler_ReturnsArgs(t *testing.T) {
	args := map[string]interface{}{
		"id":          "daemon_port1",
		"daemon_port": float64(8080),
	}
	proc := &process.Process{Args: []interface{}{args}}
	result := DaemonPortReadyHandler(proc)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestAllHandlers_EmptyArgs(t *testing.T) {
	handlers := map[string]func(*process.Process) interface{}{
		"yao_job_start":         JobStartHandler,
		"yao_job_list":          JobListHandler,
		"yao_job_get":           JobGetHandler,
		"yao_job_output":        JobOutputHandler,
		"yao_job_wait":          JobWaitHandler,
		"yao_job_stop":          JobStopHandler,
		"yao_job_settled":       JobSettledHandler,
		"yao_daemon_start":      DaemonStartHandler,
		"yao_daemon_list":       DaemonListHandler,
		"yao_daemon_status":     DaemonStatusHandler,
		"yao_daemon_stop":       DaemonStopHandler,
		"yao_daemon_restart":    DaemonRestartHandler,
		"yao_daemon_port_ready": DaemonPortReadyHandler,
	}
	for name, h := range handlers {
		proc := &process.Process{Args: []interface{}{map[string]interface{}{}}}
		result := h(proc)
		if result == nil {
			t.Errorf("%s: expected non-nil result", name)
		}
	}
}
