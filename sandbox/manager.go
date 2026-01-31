package sandbox

import (
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/yaoapp/yao/sandbox/ipc"
)

// execReadCloser wraps a Reader with a Closer
type execReadCloser struct {
	*bufio.Reader
	closer io.Closer
}

func (e *execReadCloser) Close() error {
	if e.closer != nil {
		return e.closer.Close()
	}
	return nil
}

// demuxReadCloser wraps Docker multiplexed stream and demuxes it to stdout only
// It uses a pipe to feed demuxed stdout to the reader
type demuxReadCloser struct {
	reader     io.Reader
	pipeReader *io.PipeReader
	pipeWriter *io.PipeWriter
	closer     io.Closer
	done       chan struct{}
	err        error
	closed     bool
	mu         sync.Mutex
}

// newDemuxReadCloser creates a new demuxed reader from Docker multiplexed stream
func newDemuxReadCloser(src io.Reader, closer io.Closer) *demuxReadCloser {
	pr, pw := io.Pipe()
	d := &demuxReadCloser{
		reader:     src,
		pipeReader: pr,
		pipeWriter: pw,
		closer:     closer,
		done:       make(chan struct{}),
	}

	// Start demux goroutine
	go func() {
		defer close(d.done)
		defer pw.Close()

		// Use stdcopy to demux stdout and stderr
		// We only care about stdout here, stderr goes to a discard writer
		_, err := stdcopy.StdCopy(pw, io.Discard, src)

		if err != nil && err != io.EOF {
			d.mu.Lock()
			d.err = err
			d.mu.Unlock()
		}
	}()

	return d
}

func (d *demuxReadCloser) Read(p []byte) (int, error) {
	return d.pipeReader.Read(p)
}

func (d *demuxReadCloser) Close() error {
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return nil
	}
	d.closed = true
	d.mu.Unlock()

	// Close the pipe writer first to signal EOF to any readers
	// This will cause pipeReader.Read() to return io.EOF
	d.pipeWriter.CloseWithError(io.EOF)

	// Close the source connection to interrupt stdcopy.StdCopy
	if d.closer != nil {
		d.closer.Close()
	}

	// Close the pipe reader to unblock any pending reads
	d.pipeReader.Close()

	// Wait for demux goroutine to finish with a timeout
	// Don't block forever if stdcopy.StdCopy is stuck
	select {
	case <-d.done:
		// Normal completion
	case <-time.After(5 * time.Second):
		// Timeout - goroutine may be stuck, but we've done cleanup
	}

	d.mu.Lock()
	err := d.err
	d.mu.Unlock()
	return err
}

// Manager manages sandbox containers
type Manager struct {
	mu           sync.Mutex     // Protects creation
	containers   sync.Map       // containerName → *Container
	running      int32          // Running container count
	ipcManager   *ipc.Manager   // IPC manager
	dockerClient *client.Client // Docker client
	config       *Config        // Configuration
}

// NewManager creates a new sandbox manager
func NewManager(config *Config) (*Manager, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Apply defaults for missing container paths
	if config.ContainerWorkDir == "" {
		config.ContainerWorkDir = "/workspace"
	}
	if config.ContainerIPCSocket == "" {
		config.ContainerIPCSocket = "/tmp/yao.sock"
	}

	// Initialize Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Ping Docker to verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := cli.Ping(ctx); err != nil {
		cli.Close()
		return nil, fmt.Errorf("%w: %v", ErrDockerNotAvailable, err)
	}

	// Ensure directories exist
	if err := os.MkdirAll(config.WorkspaceRoot, 0755); err != nil {
		cli.Close()
		return nil, fmt.Errorf("failed to create workspace directory: %w", err)
	}
	if err := os.MkdirAll(config.IPCDir, 0755); err != nil {
		cli.Close()
		return nil, fmt.Errorf("failed to create IPC directory: %w", err)
	}

	m := &Manager{
		dockerClient: cli,
		config:       config,
		ipcManager:   ipc.NewManager(config.IPCDir),
	}

	// Start cleanup loop
	go m.startCleanupLoop(context.Background())

	return m, nil
}

// Close closes the manager and cleans up resources
func (m *Manager) Close() error {
	m.ipcManager.CloseAll()
	return m.dockerClient.Close()
}

// GetOrCreate returns existing container or creates new one
func (m *Manager) GetOrCreate(ctx context.Context, userID, chatID string) (*Container, error) {
	name := containerName(userID, chatID)

	// Check if container already exists (fast path)
	if c, ok := m.containers.Load(name); ok {
		cont := c.(*Container)
		cont.LastUsedAt = time.Now()
		// Ensure IPC session exists (may have been closed)
		m.ensureIPCSession(ctx, userID, chatID)
		return cont, nil
	}

	// Use mutex for creation to avoid race condition
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring lock
	if c, ok := m.containers.Load(name); ok {
		cont := c.(*Container)
		cont.LastUsedAt = time.Now()
		// Ensure IPC session exists (may have been closed)
		m.ensureIPCSession(ctx, userID, chatID)
		return cont, nil
	}

	// Check running container limit
	if m.running >= int32(m.config.MaxContainers) {
		return nil, ErrTooManyContainers
	}

	// Create new container
	cont, err := m.createContainer(ctx, userID, chatID)
	if err != nil {
		return nil, err
	}

	// Store and increment counter
	m.containers.Store(name, cont)
	m.running++

	return cont, nil
}

// createContainer creates a new Docker container
func (m *Manager) createContainer(ctx context.Context, userID, chatID string) (*Container, error) {
	name := containerName(userID, chatID)

	// Ensure image exists, pull if not
	if err := m.ensureImage(ctx, m.config.Image); err != nil {
		return nil, err
	}

	// Workspace directory
	workspaceHost := filepath.Join(m.config.WorkspaceRoot, userID, chatID)
	if err := os.MkdirAll(workspaceHost, 0755); err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	// Create IPC session BEFORE container creation
	// This creates the socket file so it can be bind mounted
	sessionID := chatID
	agentCtx := &ipc.AgentContext{UserID: userID, ChatID: chatID}
	if _, err := m.ipcManager.Create(ctx, sessionID, agentCtx, nil); err != nil {
		return nil, fmt.Errorf("failed to create IPC session: %w", err)
	}

	// Get socket path (uses hash to avoid path length issues)
	ipcSocketHost := m.ipcManager.GetSocketPath(sessionID)

	// Container configuration
	containerConfig := &container.Config{
		Image:      m.config.Image,
		Cmd:        []string{"sleep", "infinity"},
		WorkingDir: m.config.ContainerWorkDir,
		User:       m.config.ContainerUser, // Empty string uses image default
		Env: []string{
			"YAO_IPC_SOCKET=" + m.config.ContainerIPCSocket,
		},
	}

	// Host configuration - mount IPC socket (now exists after ipcManager.Create)
	binds := []string{
		workspaceHost + ":" + m.config.ContainerWorkDir,
		ipcSocketHost + ":" + m.config.ContainerIPCSocket,
	}

	hostConfig := &container.HostConfig{
		Binds: binds,
		Resources: container.Resources{
			Memory:   parseMemory(m.config.MaxMemory),
			NanoCPUs: int64(m.config.MaxCPU * 1e9),
		},
		SecurityOpt: []string{"no-new-privileges"},
		CapDrop:     []string{"ALL"},
	}

	// Create container
	resp, err := m.dockerClient.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, name)
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	return &Container{
		ID:         resp.ID,
		Name:       name,
		UserID:     userID,
		ChatID:     chatID,
		Status:     StatusCreated,
		CreatedAt:  time.Now(),
		LastUsedAt: time.Now(),
	}, nil
}

// ensureImage ensures the image exists locally, pulls if not
func (m *Manager) ensureImage(ctx context.Context, imageName string) error {
	// Check if image exists locally
	_, _, err := m.dockerClient.ImageInspectWithRaw(ctx, imageName)
	if err == nil {
		return nil // Image exists
	}

	// Image not found, pull it
	reader, err := m.dockerClient.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", imageName, err)
	}
	defer reader.Close()

	// Wait for pull to complete by reading the response
	_, err = io.Copy(io.Discard, reader)
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", imageName, err)
	}

	return nil
}

// ensureRunning ensures the container is running
func (m *Manager) ensureRunning(ctx context.Context, name string) error {
	c, ok := m.containers.Load(name)
	if !ok {
		return ErrContainerNotFound
	}
	cont := c.(*Container)

	if cont.Status == StatusRunning {
		return nil
	}

	// Start the container
	if err := m.dockerClient.ContainerStart(ctx, cont.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	// Wait for container to be ready (inspect until running)
	for i := 0; i < 30; i++ {
		info, err := m.dockerClient.ContainerInspect(ctx, cont.ID)
		if err != nil {
			return fmt.Errorf("failed to inspect container: %w", err)
		}
		if info.State.Running {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Fix IPC socket permissions inside container
	// This is needed because macOS Docker Desktop doesn't properly preserve
	// Unix socket permissions when bind mounting from host
	m.fixIPCSocketPermissions(ctx, cont.ID)

	m.mu.Lock()
	cont.Status = StatusRunning
	cont.LastUsedAt = time.Now()
	m.mu.Unlock()

	return nil
}

// Stream executes command and returns stdout reader
func (m *Manager) Stream(ctx context.Context, name string, cmd []string, opts *ExecOptions) (io.ReadCloser, error) {
	// Ensure container is running
	if err := m.ensureRunning(ctx, name); err != nil {
		return nil, err
	}

	// Get container
	c, ok := m.containers.Load(name)
	if !ok {
		return nil, ErrContainerNotFound
	}
	cont := c.(*Container)

	// Update last used time
	cont.LastUsedAt = time.Now()

	// Default options
	if opts == nil {
		opts = &ExecOptions{}
	}
	if opts.WorkDir == "" {
		opts.WorkDir = "/workspace"
	}

	// Create exec instance
	execConfig := container.ExecOptions{
		Cmd:          cmd,
		WorkingDir:   opts.WorkDir,
		Env:          mapToSlice(opts.Env),
		AttachStdout: true,
		AttachStderr: true,
		AttachStdin:  opts.Stdin != nil,
	}

	execResp, err := m.dockerClient.ContainerExecCreate(ctx, cont.ID, execConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create exec: %w", err)
	}

	// Attach to exec
	attachResp, err := m.dockerClient.ContainerExecAttach(ctx, execResp.ID, container.ExecStartOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to attach to exec: %w", err)
	}

	// Handle stdin if provided
	if opts.Stdin != nil {
		go func() {
			io.Copy(attachResp.Conn, opts.Stdin)
			attachResp.CloseWrite()
		}()
	}

	// Return demuxed reader that properly handles Docker multiplexed stream
	// This removes the 8-byte header from each frame and separates stdout from stderr
	return newDemuxReadCloser(attachResp.Reader, attachResp.Conn), nil
}

// Exec executes command and waits for completion
func (m *Manager) Exec(ctx context.Context, name string, cmd []string, opts *ExecOptions) (*ExecResult, error) {
	if opts == nil {
		opts = &ExecOptions{}
	}

	// Apply timeout if specified
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	// Ensure container is running
	if err := m.ensureRunning(ctx, name); err != nil {
		return nil, err
	}

	// Get container
	c, ok := m.containers.Load(name)
	if !ok {
		return nil, ErrContainerNotFound
	}
	cont := c.(*Container)

	// Update last used time
	cont.LastUsedAt = time.Now()

	// Default options
	if opts.WorkDir == "" {
		opts.WorkDir = m.config.ContainerWorkDir
	}

	// Create exec instance
	execConfig := container.ExecOptions{
		Cmd:          cmd,
		WorkingDir:   opts.WorkDir,
		Env:          mapToSlice(opts.Env),
		AttachStdout: true,
		AttachStderr: true,
		AttachStdin:  opts.Stdin != nil,
	}

	execResp, err := m.dockerClient.ContainerExecCreate(ctx, cont.ID, execConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create exec: %w", err)
	}

	// Attach to exec
	attachResp, err := m.dockerClient.ContainerExecAttach(ctx, execResp.ID, container.ExecStartOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to attach to exec: %w", err)
	}
	defer attachResp.Close()

	// Handle stdin if provided
	if opts.Stdin != nil {
		go func() {
			io.Copy(attachResp.Conn, opts.Stdin)
			attachResp.CloseWrite()
		}()
	}

	// Read output with context awareness
	outputCh := make(chan []byte, 1)
	errCh := make(chan error, 1)

	// Buffers for demuxed stdout and stderr
	var stdoutBuf, stderrBuf bytes.Buffer

	go func() {
		// Use stdcopy to properly demux Docker multiplexed stream
		_, err := stdcopy.StdCopy(&stdoutBuf, &stderrBuf, attachResp.Reader)
		if err != nil && err != io.EOF {
			errCh <- err
			return
		}
		outputCh <- nil
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-errCh:
		return nil, fmt.Errorf("failed to read output: %w", err)
	case <-outputCh:
		// Output received
	}

	// Wait for exec to complete and get exit code
	var exitCode int
	for i := 0; i < 100; i++ { // Max 10 seconds wait
		inspect, err := m.dockerClient.ContainerExecInspect(ctx, execResp.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to inspect exec: %w", err)
		}
		if !inspect.Running {
			exitCode = inspect.ExitCode
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	return &ExecResult{
		ExitCode: exitCode,
		Stdout:   stdoutBuf.String(),
		Stderr:   stderrBuf.String(),
	}, nil
}

// Start starts a stopped container
func (m *Manager) Start(ctx context.Context, name string) error {
	return m.ensureRunning(ctx, name)
}

// Stop stops container but preserves data
func (m *Manager) Stop(ctx context.Context, name string) error {
	c, ok := m.containers.Load(name)
	if !ok {
		return nil
	}
	cont := c.(*Container)

	// Only stop if running
	if cont.Status != StatusRunning {
		return nil
	}

	if err := m.dockerClient.ContainerStop(ctx, cont.ID, container.StopOptions{}); err != nil {
		// Ignore "not running" error
		if !strings.Contains(err.Error(), "is not running") {
			return fmt.Errorf("failed to stop container: %w", err)
		}
	}

	// Update status, decrement running count
	m.mu.Lock()
	if cont.Status == StatusRunning {
		cont.Status = StatusStopped
		m.running--
	}
	m.mu.Unlock()

	return nil
}

// Remove deletes container and its data
func (m *Manager) Remove(ctx context.Context, name string) error {
	// Stop first if running
	m.Stop(ctx, name)

	c, ok := m.containers.Load(name)
	if !ok {
		return nil
	}
	cont := c.(*Container)

	// Close IPC session
	m.ipcManager.Close(cont.ChatID)

	if err := m.dockerClient.ContainerRemove(ctx, cont.ID, container.RemoveOptions{Force: true}); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	// Remove from map
	m.containers.Delete(name)

	return nil
}

// KillProcess kills a process inside the container by name pattern
// This is used to forcefully stop long-running processes like Claude CLI
func (m *Manager) KillProcess(ctx context.Context, name string, processPattern string) error {
	c, ok := m.containers.Load(name)
	if !ok {
		return ErrContainerNotFound
	}
	cont := c.(*Container)

	// Use pkill to kill processes matching the pattern
	// -f matches against the full command line
	// Use SIGKILL (-9) to ensure the process is killed immediately
	cmd := []string{"pkill", "-9", "-f", processPattern}

	execConfig := container.ExecOptions{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
	}

	execResp, err := m.dockerClient.ContainerExecCreate(ctx, cont.ID, execConfig)
	if err != nil {
		return fmt.Errorf("failed to create exec for kill: %w", err)
	}

	// Start the exec
	if err := m.dockerClient.ContainerExecStart(ctx, execResp.ID, container.ExecStartOptions{}); err != nil {
		return fmt.Errorf("failed to start exec for kill: %w", err)
	}

	return nil
}

// List returns all containers for a user
func (m *Manager) List(ctx context.Context, userID string) ([]*Container, error) {
	var result []*Container
	prefix := fmt.Sprintf("yao-sandbox-%s-", userID)

	m.containers.Range(func(key, value interface{}) bool {
		name := key.(string)
		if strings.HasPrefix(name, prefix) {
			result = append(result, value.(*Container))
		}
		return true
	})

	return result, nil
}

// Cleanup stops idle containers
func (m *Manager) Cleanup(ctx context.Context) error {
	now := time.Now()

	m.containers.Range(func(key, value interface{}) bool {
		name := key.(string)
		c := value.(*Container)

		// Stop idle containers
		if c.Status == StatusRunning && now.Sub(c.LastUsedAt) > m.config.IdleTimeout {
			m.Stop(ctx, name)
		}

		return true
	})

	return nil
}

// startCleanupLoop starts the periodic cleanup loop
func (m *Manager) startCleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.Cleanup(ctx)
		}
	}
}

// WriteFile writes content to a file in container
func (m *Manager) WriteFile(ctx context.Context, name, path string, content []byte) error {
	c, ok := m.containers.Load(name)
	if !ok {
		return ErrContainerNotFound
	}
	cont := c.(*Container)

	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if dir != "/" && dir != "." {
		result, err := m.Exec(ctx, name, []string{"mkdir", "-p", dir}, nil)
		if err != nil {
			return fmt.Errorf("failed to create parent directory: %w", err)
		}
		if result.ExitCode != 0 {
			return fmt.Errorf("mkdir failed with exit code %d: %s", result.ExitCode, result.Stdout)
		}

		// Verify directory was created
		verifyResult, err := m.Exec(ctx, name, []string{"test", "-d", dir}, nil)
		if err != nil {
			return fmt.Errorf("failed to verify directory: %w", err)
		}
		if verifyResult.ExitCode != 0 {
			return fmt.Errorf("directory %s was not created", dir)
		}
	}

	// Create a tar archive with the file
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	hdr := &tar.Header{
		Name: filepath.Base(path),
		Mode: 0644,
		Size: int64(len(content)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	if _, err := tw.Write(content); err != nil {
		return err
	}
	if err := tw.Close(); err != nil {
		return err
	}

	// Copy to container
	return m.dockerClient.CopyToContainer(ctx, cont.ID, dir, &buf, container.CopyToContainerOptions{})
}

// ReadFile reads content from a file in container
func (m *Manager) ReadFile(ctx context.Context, name, path string) ([]byte, error) {
	c, ok := m.containers.Load(name)
	if !ok {
		return nil, ErrContainerNotFound
	}
	cont := c.(*Container)

	reader, _, err := m.dockerClient.CopyFromContainer(ctx, cont.ID, path)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	// Extract from tar
	tr := tar.NewReader(reader)
	_, err = tr.Next()
	if err != nil {
		return nil, err
	}

	return io.ReadAll(tr)
}

// ListDir lists directory contents in container
func (m *Manager) ListDir(ctx context.Context, name, path string) ([]FileInfo, error) {
	result, err := m.Exec(ctx, name, []string{"ls", "-la", "--time-style=+%s", path}, nil)
	if err != nil {
		return nil, err
	}

	return parseLS(result.Stdout), nil
}

// Stat returns file info
func (m *Manager) Stat(ctx context.Context, name, path string) (*FileInfo, error) {
	result, err := m.Exec(ctx, name, []string{"stat", "--format=%n|%s|%f|%Y|%F", path}, nil)
	if err != nil {
		return nil, err
	}
	return parseStat(result.Stdout), nil
}

// MkDir creates directory in container
func (m *Manager) MkDir(ctx context.Context, name, path string) error {
	_, err := m.Exec(ctx, name, []string{"mkdir", "-p", path}, nil)
	return err
}

// RemoveFile removes file or directory in container
func (m *Manager) RemoveFile(ctx context.Context, name, path string) error {
	_, err := m.Exec(ctx, name, []string{"rm", "-rf", path}, nil)
	return err
}

// CopyToContainer copies from host to container
func (m *Manager) CopyToContainer(ctx context.Context, name, hostPath, containerPath string) error {
	c, ok := m.containers.Load(name)
	if !ok {
		return ErrContainerNotFound
	}
	cont := c.(*Container)

	// Create tar archive from host path
	archive, err := createTarFromPath(hostPath)
	if err != nil {
		return err
	}
	defer archive.Close()

	return m.dockerClient.CopyToContainer(ctx, cont.ID, containerPath, archive, container.CopyToContainerOptions{})
}

// CopyFromContainer copies from container to host
func (m *Manager) CopyFromContainer(ctx context.Context, name, containerPath, hostPath string) error {
	c, ok := m.containers.Load(name)
	if !ok {
		return ErrContainerNotFound
	}
	cont := c.(*Container)

	reader, _, err := m.dockerClient.CopyFromContainer(ctx, cont.ID, containerPath)
	if err != nil {
		return err
	}
	defer reader.Close()

	return extractTarToPath(reader, hostPath)
}

// GetIPCManager returns the IPC manager
func (m *Manager) GetIPCManager() *ipc.Manager {
	return m.ipcManager
}

// GetConfig returns the configuration
func (m *Manager) GetConfig() *Config {
	return m.config
}

// ensureIPCSession ensures IPC session exists for the given chatID
// This is called when reusing an existing container to handle cases where
// the IPC session was closed but the container still exists
func (m *Manager) ensureIPCSession(ctx context.Context, userID, chatID string) {
	sessionID := chatID
	// Check if session already exists
	if _, ok := m.ipcManager.Get(sessionID); ok {
		return
	}
	// Create new session (ignore error - container can work without IPC)
	agentCtx := &ipc.AgentContext{UserID: userID, ChatID: chatID}
	m.ipcManager.Create(ctx, sessionID, agentCtx, nil)
}

// fixIPCSocketPermissions fixes IPC socket permissions inside the container
// This is needed because macOS Docker Desktop with gRPC-FUSE doesn't properly
// preserve Unix socket permissions when bind mounting from host.
// We run chmod as root (using container exec with User override) to make the
// socket accessible to the sandbox user.
func (m *Manager) fixIPCSocketPermissions(ctx context.Context, containerID string) {
	// Execute chmod as root to fix socket permissions
	execConfig := container.ExecOptions{
		Cmd:  []string{"chmod", "666", m.config.ContainerIPCSocket},
		User: "root", // Run as root to be able to change permissions
	}

	execResp, err := m.dockerClient.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		// Log but don't fail - container can work without proper IPC
		return
	}

	// Start the exec and wait for completion
	err = m.dockerClient.ContainerExecStart(ctx, execResp.ID, container.ExecStartOptions{})
	if err != nil {
		// Log but don't fail
		return
	}

	// Wait briefly for the chmod to complete
	time.Sleep(50 * time.Millisecond)
}
