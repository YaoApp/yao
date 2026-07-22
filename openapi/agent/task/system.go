package task

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/agent/computer"
	lifecycletypes "github.com/yaoapp/yao/agent/sandbox/v2/types"
	tasksvc "github.com/yaoapp/yao/agent/task"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	"github.com/yaoapp/yao/openapi/response"
	sandbox "github.com/yaoapp/yao/sandbox/v2"
)

// handleTaskPortsGet returns listening ports for the sandbox associated with this task.
func handleTaskPortsGet(c *gin.Context) {
	computer, err := resolveComputer(c)
	if err != nil {
		return
	}
	if computer == nil {
		response.RespondWithSuccess(c, http.StatusOK, map[string]interface{}{
			"status":  "sandbox_not_running",
			"message": "sandbox is not running",
		})
		return
	}

	ports, err := computer.ListPorts(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, map[string]interface{}{
		"ports": ports,
	})
}

// handleTaskProcessesGet returns running processes for the sandbox associated with this task.
func handleTaskProcessesGet(c *gin.Context) {
	computer, err := resolveComputer(c)
	if err != nil {
		return
	}
	if computer == nil {
		response.RespondWithSuccess(c, http.StatusOK, map[string]interface{}{
			"status":  "sandbox_not_running",
			"message": "sandbox is not running",
		})
		return
	}

	var opts []sandbox.ListProcessesOption
	if c.Query("fast") == "true" {
		opts = append(opts, sandbox.WithSkipCPU())
	}

	procs, load, err := computer.ListProcesses(c.Request.Context(), opts...)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	result := map[string]interface{}{
		"processes": procs,
	}
	if load != nil {
		result["load"] = load
	}
	response.RespondWithSuccess(c, http.StatusOK, result)
}

// handleTaskExecPost executes a command in the sandbox associated with this task.
func handleTaskExecPost(c *gin.Context) {
	computer, err := resolveComputer(c)
	if err != nil {
		return
	}
	if computer == nil {
		respondError(c, http.StatusServiceUnavailable, fmt.Errorf("sandbox is not running"))
		return
	}

	var req struct {
		Cmd  []string `json:"cmd" binding:"required"`
		Root bool     `json:"root"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}

	var result *sandbox.ExecResult
	if req.Root {
		if box, ok := computer.(*sandbox.Box); ok {
			result, err = box.RootExec(c.Request.Context(), req.Cmd)
		} else {
			result, err = computer.Exec(c.Request.Context(), req.Cmd)
		}
	} else {
		result, err = computer.Exec(c.Request.Context(), req.Cmd)
	}
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}

	response.RespondWithSuccess(c, http.StatusOK, map[string]interface{}{
		"exit_code": result.ExitCode,
		"stdout":    result.Stdout,
		"stderr":    result.Stderr,
	})
}

// resolveComputer finds the Computer (Box or Host) for the current task via
// the unified computer.Lookup path.
// Returns nil Computer + nil error when sandbox is not running (Box mode only).
// Host mode always returns a valid Computer (no "not running" state).
// On auth/lookup errors it writes the error response and returns non-nil error.
func resolveComputer(c *gin.Context) (sandbox.Computer, error) {
	auth := authorized.GetInfo(c)
	chatID := c.Param("chat_id")

	pauth := toProcessAuth(auth)
	task, err := tasksvc.Get(c.Request.Context(), pauth, chatID)
	if err != nil {
		respondError(c, http.StatusNotFound, err)
		return nil, err
	}

	ast, astErr := assistant.Get(task.AssistantID)
	if astErr != nil || ast == nil {
		respondError(c, http.StatusNotFound, fmt.Errorf("assistant %q not found", task.AssistantID))
		return nil, fmt.Errorf("assistant not found")
	}

	cfg := ast.SandboxV2
	if cfg == nil {
		cfg = &lifecycletypes.SandboxConfig{}
	}

	wsID := ""
	if task.LastWorkspace != nil {
		wsID = *task.LastWorkspace
	}

	comp, err := computer.Lookup(c.Request.Context(), &computer.LookupOpts{
		Auth:        auth,
		AssistantID: task.AssistantID,
		WorkspaceID: wsID,
		ChatID:      chatID,
		Image:       cfg.Computer.Image,
		Lifecycle:   cfg.Lifecycle,
	})
	if err != nil {
		respondError(c, http.StatusServiceUnavailable, err)
		return nil, err
	}
	return comp, nil
}
