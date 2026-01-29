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
		return cont, nil
	}

	// Use mutex for creation to avoid race condition
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring lock
	if c, ok := m.containers.Load(name); ok {
		cont := c.(*Container)
		cont.LastUsedAt = time.Now()
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

	// IPC socket path
	sessionID := chatID
	ipcSocketHost := filepath.Join(m.config.IPCDir, sessionID+".sock")

	// Container configuration
	containerConfig := &container.Config{
		Image:      m.config.Image,
		Cmd:        []string{"sleep", "infinity"},
		WorkingDir: "/workspace",
		Env: []string{
			"YAO_IPC_SOCKET=/tmp/yao.sock",
		},
	}

	// Host configuration
	hostConfig := &container.HostConfig{
		Binds: []string{
			workspaceHost + ":/workspace",
			ipcSocketHost + ":/tmp/yao.sock",
		},
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

	// Wrap in a ReadCloser
	return &execReadCloser{
		Reader: attachResp.Reader,
		closer: attachResp.Conn,
	}, nil
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

	reader, err := m.Stream(ctx, name, cmd, opts)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	// Read output with context awareness
	outputCh := make(chan []byte, 1)
	errCh := make(chan error, 1)

	go func() {
		output, err := io.ReadAll(reader)
		if err != nil {
			errCh <- err
			return
		}
		outputCh <- output
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-errCh:
		return nil, fmt.Errorf("failed to read output: %w", err)
	case output := <-outputCh:
		// Parse Docker multiplexed stream
		// TODO: Properly demux stdout/stderr from Docker stream
		stdout := string(output)

		return &ExecResult{
			ExitCode: 0,
			Stdout:   stdout,
			Stderr:   "",
		}, nil
	}
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
		if _, err := m.Exec(ctx, name, []string{"mkdir", "-p", dir}, nil); err != nil {
			return fmt.Errorf("failed to create parent directory: %w", err)
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
