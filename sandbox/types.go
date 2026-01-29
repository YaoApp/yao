package sandbox

import (
	"io"
	"os"
	"time"

	"github.com/yaoapp/yao/sandbox/ipc"
)

// Container represents a sandbox container
type Container struct {
	ID         string       // Docker container ID
	Name       string       // Container name: yao-sandbox-{userID}-{chatID}
	UserID     string       // User identifier
	ChatID     string       // Chat/session identifier
	Status     string       // created, running, stopped
	CreatedAt  time.Time    // Container creation time
	LastUsedAt time.Time    // Last activity time
	IPCSession *ipc.Session // Associated IPC session
}

// ExecOptions configures command execution
type ExecOptions struct {
	WorkDir string            // Working directory inside container
	Env     map[string]string // Environment variables
	Stdin   io.Reader         // Standard input
	Timeout time.Duration     // Execution timeout (0 = no timeout)
}

// ExecResult contains the result of command execution
type ExecResult struct {
	ExitCode int    // Exit code
	Stdout   string // Standard output
	Stderr   string // Standard error
}

// FileInfo represents file metadata
type FileInfo struct {
	Name    string      // File name
	Path    string      // Full path
	Size    int64       // Size in bytes
	Mode    os.FileMode // File mode
	ModTime time.Time   // Modification time
	IsDir   bool        // Is directory
}

// ContainerStatus constants
const (
	StatusCreated = "created"
	StatusRunning = "running"
	StatusStopped = "stopped"
)
