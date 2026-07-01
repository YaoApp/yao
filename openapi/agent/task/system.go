package task

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/agent/assistant"
	lifecycle "github.com/yaoapp/yao/agent/sandbox/v2"
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

// resolveComputer finds the Computer (Box or Host) for the current task.
// Returns nil Computer + nil error when sandbox is not running (Box mode only).
// Host mode always returns a valid Computer (no "not running" state).
// On auth/lookup errors it writes the error response and returns non-nil error.
func resolveComputer(c *gin.Context) (sandbox.Computer, error) {
	auth := toProcessAuth(authorized.GetInfo(c))
	chatID := c.Param("chat_id")

	task, err := tasksvc.Get(c.Request.Context(), auth, chatID)
	if err != nil {
		respondError(c, http.StatusNotFound, err)
		return nil, err
	}

	mgr := sandbox.M()

	ast, astErr := assistant.Get(task.AssistantID)
	if astErr != nil || ast == nil {
		respondError(c, http.StatusNotFound, fmt.Errorf("assistant %q not found", task.AssistantID))
		return nil, fmt.Errorf("assistant not found")
	}

	cfg := ast.SandboxV2
	if cfg == nil {
		cfg = &lifecycletypes.SandboxConfig{}
	}

	// Determine if this is Host mode (no container image) or Box mode.
	isHostMode := !ast.HasSandboxV2() || cfg.Computer.Image == ""

	if isHostMode {
		// Host mode: always available, no "not running" state.
		// Use the local node (Host mode only needs HostExec + SystemQuery).
		host, err := mgr.Host(c.Request.Context(), "local")
		if err != nil {
			respondError(c, http.StatusInternalServerError, fmt.Errorf("host mode unavailable: %w", err))
			return nil, err
		}
		return host, nil
	}

	// Box mode: look up existing container by identifier.
	ownerID := auth.UserID
	identifier := lifecycle.BuildIdentifier(cfg, ownerID, chatID, task.AssistantID, "", nil)
	if identifier == "" {
		return nil, nil
	}

	box, err := mgr.Get(c.Request.Context(), identifier)
	if err != nil || box == nil {
		return nil, nil
	}
	if box.IsStopped() {
		return nil, nil
	}

	return box, nil
}
