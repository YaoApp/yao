package vncproxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/gorilla/websocket"
)

// ipCacheEntry holds cached container IP with expiration
type ipCacheEntry struct {
	IP        string
	ExpiresAt time.Time
}

// Proxy handles VNC proxy requests
type Proxy struct {
	config       *Config
	dockerClient *client.Client
	ipCache      sync.Map // containerName -> *ipCacheEntry
	upgrader     websocket.Upgrader
}

// NewProxy creates a new VNC proxy
func NewProxy(config *Config) (*Proxy, error) {
	if config == nil {
		config = DefaultConfig()
	}
	config.Init()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Verify Docker connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := cli.Ping(ctx); err != nil {
		cli.Close()
		return nil, fmt.Errorf("Docker not available: %w", err)
	}

	return &Proxy{
		config:       config,
		dockerClient: cli,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for VNC
			},
			Subprotocols: []string{"binary"}, // noVNC uses binary subprotocol
		},
	}, nil
}

// Close closes the proxy and releases resources
func (p *Proxy) Close() error {
	return p.dockerClient.Close()
}

// extractSandboxID extracts sandbox ID from request path
// Expected format: /v1/sandbox/{id}/vnc/...
func extractSandboxID(r *http.Request) string {
	path := r.URL.Path
	// Remove prefix /v1/sandbox/
	path = strings.TrimPrefix(path, "/v1/sandbox/")
	// Get ID (first segment before next /)
	if idx := strings.Index(path, "/"); idx > 0 {
		return path[:idx]
	}
	return path
}

// HandleVNCStatus returns VNC status for a container
// GET /v1/sandbox/{id}/vnc
func (p *Proxy) HandleVNCStatus(w http.ResponseWriter, r *http.Request) {
	sandboxID := extractSandboxID(r)
	containerName := p.config.ContainerNamePrefix + sandboxID

	response := map[string]interface{}{
		"sandbox_id": sandboxID,
		"container":  containerName,
	}

	// Check if container exists and is running
	_, err := p.getContainerIP(r.Context(), containerName)
	if err != nil {
		response["available"] = false
		response["status"] = "unavailable"
		response["message"] = "Container not available"
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Check if VNC is enabled for this container
	if !p.checkVNCEnabled(r.Context(), containerName) {
		response["available"] = false
		response["status"] = "not_supported"
		response["message"] = "VNC not available for this container type"
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Check if VNC services are ready (try to connect to websockify port)
	if !p.checkVNCReady(r.Context(), containerName) {
		response["available"] = false
		response["status"] = "starting"
		response["message"] = "VNC services are starting..."
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// VNC is ready
	response["available"] = true
	response["status"] = "ready"
	response["client_url"] = fmt.Sprintf("/v1/sandbox/%s/vnc/client", sandboxID)
	response["websocket_url"] = fmt.Sprintf("/v1/sandbox/%s/vnc/ws", sandboxID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleVNCClient serves the noVNC client page
// GET /v1/sandbox/{id}/vnc/client?viewonly=true|false
func (p *Proxy) HandleVNCClient(w http.ResponseWriter, r *http.Request) {
	sandboxID := extractSandboxID(r)
	containerName := p.config.ContainerNamePrefix + sandboxID

	// Verify container exists and is running
	_, err := p.getContainerIP(r.Context(), containerName)
	if err != nil {
		http.Error(w, "Container not available", http.StatusNotFound)
		return
	}

	if !p.checkVNCEnabled(r.Context(), containerName) {
		http.Error(w, "VNC not available for this container", http.StatusBadRequest)
		return
	}

	// Get viewonly parameter (default: false = interactive)
	viewOnly := r.URL.Query().Get("viewonly") == "true"

	// Serve inline noVNC HTML page with status checking
	wsPath := fmt.Sprintf("/v1/sandbox/%s/vnc/ws", sandboxID)
	p.serveNoVNCPage(w, sandboxID, wsPath, viewOnly)
}

// HandleVNCWebSocket proxies WebSocket connection to container VNC
// GET /v1/sandbox/{id}/vnc/ws
func (p *Proxy) HandleVNCWebSocket(w http.ResponseWriter, r *http.Request) {
	sandboxID := extractSandboxID(r)
	containerName := p.config.ContainerNamePrefix + sandboxID

	// Get VNC endpoint (uses port mapping if available, otherwise container IP)
	targetAddr, err := p.getVNCEndpoint(r.Context(), containerName)
	if err != nil {
		http.Error(w, "Container not available", http.StatusNotFound)
		return
	}

	// Upgrade HTTP to WebSocket
	clientConn, err := p.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return // Upgrader already sent error response
	}
	defer clientConn.Close()

	// Connect to container's websockify
	targetConn, err := net.DialTimeout("tcp", targetAddr, 5*time.Second)
	if err != nil {
		clientConn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "VNC connection failed"))
		return
	}
	defer targetConn.Close()

	// Bidirectional proxy
	done := make(chan struct{})

	// Client -> Container
	go func() {
		defer func() { done <- struct{}{} }()
		for {
			messageType, data, err := clientConn.ReadMessage()
			if err != nil {
				return
			}
			if messageType == websocket.BinaryMessage {
				if _, err := targetConn.Write(data); err != nil {
					return
				}
			}
		}
	}()

	// Container -> Client
	go func() {
		defer func() { done <- struct{}{} }()
		buf := make([]byte, 32*1024)
		for {
			n, err := targetConn.Read(buf)
			if err != nil {
				return
			}
			if err := clientConn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
				return
			}
		}
	}()

	// Wait for either direction to close
	<-done
}

// getContainerIP gets the IP address of a container, using cache with TTL
func (p *Proxy) getContainerIP(ctx context.Context, containerName string) (string, error) {
	// Check cache
	if cached, ok := p.ipCache.Load(containerName); ok {
		entry := cached.(*ipCacheEntry)
		if time.Now().Before(entry.ExpiresAt) {
			return entry.IP, nil
		}
		// Cache expired, delete it
		p.ipCache.Delete(containerName)
	}

	// Get from Docker
	info, err := p.dockerClient.ContainerInspect(ctx, containerName)
	if err != nil {
		return "", fmt.Errorf("container not found: %w", err)
	}

	if !info.State.Running {
		return "", fmt.Errorf("container not running")
	}

	// Get IP from the specified network or default bridge
	var ip string
	if info.NetworkSettings != nil && info.NetworkSettings.Networks != nil {
		if net, ok := info.NetworkSettings.Networks[p.config.DockerNetwork]; ok {
			ip = net.IPAddress
		} else {
			// Try to get IP from any network
			for _, net := range info.NetworkSettings.Networks {
				if net.IPAddress != "" {
					ip = net.IPAddress
					break
				}
			}
		}
	}

	if ip == "" {
		return "", fmt.Errorf("container has no IP address")
	}

	// Cache the result
	p.ipCache.Store(containerName, &ipCacheEntry{
		IP:        ip,
		ExpiresAt: time.Now().Add(p.config.IPCacheTTL),
	})

	return ip, nil
}

// getVNCEndpoint returns the host:port to connect to for VNC
// It first checks for port mapping (for Docker Desktop), then falls back to container IP
func (p *Proxy) getVNCEndpoint(ctx context.Context, containerName string) (string, error) {
	info, err := p.dockerClient.ContainerInspect(ctx, containerName)
	if err != nil {
		return "", fmt.Errorf("container not found: %w", err)
	}

	if !info.State.Running {
		return "", fmt.Errorf("container not running")
	}

	// Check for port mapping first (for Docker Desktop on macOS/Windows)
	if info.NetworkSettings != nil && info.NetworkSettings.Ports != nil {
		portKey := nat.Port(fmt.Sprintf("%d/tcp", p.config.ContainerNoVNCPort))
		if bindings, ok := info.NetworkSettings.Ports[portKey]; ok && len(bindings) > 0 {
			binding := bindings[0]
			if binding.HostPort != "" {
				// Use mapped port on localhost
				host := binding.HostIP
				if host == "" || host == "0.0.0.0" {
					host = "127.0.0.1"
				}
				return net.JoinHostPort(host, binding.HostPort), nil
			}
		}
	}

	// Fall back to container IP (works on Linux with native Docker)
	ip, err := p.getContainerIP(ctx, containerName)
	if err != nil {
		return "", err
	}
	return net.JoinHostPort(ip, fmt.Sprintf("%d", p.config.ContainerNoVNCPort)), nil
}

// checkVNCEnabled checks if container has VNC enabled by checking env vars
func (p *Proxy) checkVNCEnabled(ctx context.Context, containerName string) bool {
	info, err := p.dockerClient.ContainerInspect(ctx, containerName)
	if err != nil {
		return false
	}

	// Check environment variables for VNC_ENABLED or SANDBOX_VNC_ENABLED
	for _, env := range info.Config.Env {
		if strings.HasPrefix(env, "SANDBOX_VNC_ENABLED=true") ||
			strings.HasPrefix(env, "VNC_ENABLED=true") {
			return true
		}
	}

	// Also check if container image is a VNC-enabled variant
	imageName := info.Config.Image
	if strings.Contains(imageName, "playwright") ||
		strings.Contains(imageName, "desktop") {
		return true
	}

	return false
}

// checkVNCReady tests if VNC services are ready
// Uses docker exec to test port connectivity (works across platforms including macOS Docker Desktop)
func (p *Proxy) checkVNCReady(ctx context.Context, containerName string) bool {
	// Use docker exec to test port connectivity from inside the container
	// This approach works regardless of host network configuration
	execConfig := container.ExecOptions{
		Cmd:          []string{"sh", "-c", fmt.Sprintf("nc -z localhost %d 2>/dev/null || (echo | timeout 1 cat < /dev/tcp/localhost/%d > /dev/null 2>&1)", p.config.ContainerNoVNCPort, p.config.ContainerNoVNCPort)},
		AttachStdout: false,
		AttachStderr: false,
	}

	execResp, err := p.dockerClient.ContainerExecCreate(ctx, containerName, execConfig)
	if err != nil {
		return false
	}

	err = p.dockerClient.ContainerExecStart(ctx, execResp.ID, container.ExecStartOptions{})
	if err != nil {
		return false
	}

	// Wait for exec to complete and check exit code
	for i := 0; i < 10; i++ {
		inspect, err := p.dockerClient.ContainerExecInspect(ctx, execResp.ID)
		if err != nil {
			return false
		}
		if !inspect.Running {
			return inspect.ExitCode == 0
		}
		time.Sleep(100 * time.Millisecond)
	}

	return false
}

// serveNoVNCPage serves an inline HTML page with noVNC client
func (p *Proxy) serveNoVNCPage(w http.ResponseWriter, sandboxID, wsPath string, viewOnly bool) {
	viewOnlyStr := "false"
	modeIndicator := "可交互"
	modeColor := "#4CAF50"
	if viewOnly {
		viewOnlyStr = "true"
		modeIndicator = "只读模式"
		modeColor = "#FF9800"
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>VNC - %s</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        html, body { width: 100%%; height: 100%%; overflow: hidden; background: #1e1e1e; }
        #loading {
            position: absolute; top: 0; left: 0; right: 0; bottom: 0;
            display: flex; flex-direction: column; align-items: center; justify-content: center;
            background: #1e1e1e; color: #fff; font-family: system-ui, sans-serif;
        }
        .spinner {
            width: 50px; height: 50px; border: 4px solid #333;
            border-top-color: #4CAF50; border-radius: 50%%;
            animation: spin 1s linear infinite; margin-bottom: 20px;
        }
        @keyframes spin { to { transform: rotate(360deg); } }
        #status { font-size: 16px; margin-bottom: 10px; }
        #retry-count { font-size: 14px; color: #888; }
        #error { color: #f44336; display: none; }
        #screen { width: 100%%; height: 100%%; display: none; }
        #mode-indicator {
            position: fixed; top: 10px; right: 10px; padding: 5px 12px;
            background: %s; color: white; border-radius: 4px;
            font-family: system-ui, sans-serif; font-size: 12px;
            z-index: 1000; opacity: 0.9;
        }
    </style>
</head>
<body>
    <div id="loading">
        <div class="spinner"></div>
        <div id="status">正在连接 VNC...</div>
        <div id="retry-count"></div>
        <div id="error"></div>
    </div>
    <div id="mode-indicator">%s</div>
    <div id="screen"></div>

    <script type="module">
        import RFB from 'https://cdn.jsdelivr.net/npm/@novnc/novnc@1.5.0/lib/rfb.js';
        
        const sandboxID = '%s';
        const wsPath = '%s';
        const viewOnly = %s;
        const statusAPI = '/v1/sandbox/' + sandboxID + '/vnc';
        const maxRetries = 30;
        let retryCount = 0;
        
        const loading = document.getElementById('loading');
        const screen = document.getElementById('screen');
        const status = document.getElementById('status');
        const retryCountEl = document.getElementById('retry-count');
        const errorEl = document.getElementById('error');
        const modeIndicator = document.getElementById('mode-indicator');
        
        async function checkStatus() {
            try {
                const res = await fetch(statusAPI);
                const data = await res.json();
                
                if (data.status === 'ready') {
                    status.textContent = '正在初始化 VNC 客户端...';
                    connectVNC();
                    return;
                }
                
                if (data.status === 'starting') {
                    status.textContent = 'VNC 服务启动中...';
                } else if (data.status === 'not_supported') {
                    showError('此容器不支持 VNC');
                    return;
                } else {
                    status.textContent = '等待容器就绪...';
                }
                
                retryCount++;
                retryCountEl.textContent = '重试 ' + retryCount + '/' + maxRetries;
                
                if (retryCount >= maxRetries) {
                    showError('连接超时，请稍后重试');
                    return;
                }
                
                setTimeout(checkStatus, 1000);
            } catch (err) {
                retryCount++;
                if (retryCount >= maxRetries) {
                    showError('无法连接到服务器');
                    return;
                }
                setTimeout(checkStatus, 1000);
            }
        }
        
        function showError(msg) {
            status.style.display = 'none';
            retryCountEl.style.display = 'none';
            document.querySelector('.spinner').style.display = 'none';
            errorEl.textContent = msg;
            errorEl.style.display = 'block';
        }
        
        function connectVNC() {
            loading.style.display = 'none';
            screen.style.display = 'block';
            
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const wsURL = protocol + '//' + window.location.host + wsPath;
            
            const rfb = new RFB(screen, wsURL);
            rfb.viewOnly = viewOnly;
            rfb.scaleViewport = true;
            rfb.resizeSession = true;
            
            rfb.addEventListener('connect', () => {
                console.log('VNC connected');
                modeIndicator.style.display = 'block';
            });
            
            rfb.addEventListener('disconnect', (e) => {
                console.log('VNC disconnected', e.detail);
                loading.style.display = 'flex';
                screen.style.display = 'none';
                modeIndicator.style.display = 'none';
                if (e.detail.clean) {
                    status.textContent = 'VNC 连接已关闭';
                } else {
                    showError('VNC 连接断开');
                }
            });
        }
        
        // Start checking status
        checkStatus();
    </script>
</body>
</html>`, sandboxID, modeColor, modeIndicator, sandboxID, wsPath, viewOnlyStr)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	io.WriteString(w, html)
}

// RegisterRoutes registers VNC proxy routes to an HTTP mux
func (p *Proxy) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v1/sandbox/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Match /v1/sandbox/{id}/vnc
		if strings.HasSuffix(path, "/vnc") {
			p.HandleVNCStatus(w, r)
			return
		}

		// Match /v1/sandbox/{id}/vnc/client
		if strings.HasSuffix(path, "/vnc/client") {
			p.HandleVNCClient(w, r)
			return
		}

		// Match /v1/sandbox/{id}/vnc/ws
		if strings.HasSuffix(path, "/vnc/ws") {
			p.HandleVNCWebSocket(w, r)
			return
		}

		http.NotFound(w, r)
	})
}

// Helper function to check if request requires VNC container
func (p *Proxy) isVNCRequest(r *http.Request) bool {
	path := r.URL.Path
	return strings.Contains(path, "/vnc")
}
