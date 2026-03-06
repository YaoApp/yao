package grpc

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const defaultHeartbeatInterval = 10 * time.Second

// HeartbeatLoop sends periodic heartbeats to the Yao gRPC server.
// It runs until ctx is cancelled. The sandboxID comes from YAO_SANDBOX_ID.
func HeartbeatLoop(ctx context.Context, client *Client, sandboxID string) {
	interval := defaultHeartbeatInterval
	if s := os.Getenv("YAO_HEARTBEAT_INTERVAL"); s != "" {
		if d, err := time.ParseDuration(s); err == nil && d > 0 {
			interval = d
		}
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cpu, mem := sampleResources()
			procs := countUserProcesses()
			action, err := client.Heartbeat(ctx, sandboxID, cpu, mem, procs)
			if err != nil {
				continue
			}
			if action == "shutdown" {
				fmt.Fprintf(os.Stderr, "yao-grpc: received shutdown signal\n")
				p, _ := os.FindProcess(os.Getpid())
				p.Signal(os.Interrupt)
				return
			}
		}
	}
}

// countUserProcesses counts running processes owned by the current user.
func countUserProcesses() int32 {
	if runtime.GOOS != "linux" {
		return 0
	}

	out, err := exec.Command("sh", "-c", "ps -e --no-headers | wc -l").Output()
	if err != nil {
		return 0
	}
	n, _ := strconv.Atoi(strings.TrimSpace(string(out)))
	return int32(n)
}

// sampleResources reads basic CPU/memory stats from /proc (Linux only).
func sampleResources() (cpuPercent int32, memBytes int64) {
	if runtime.GOOS != "linux" {
		return 0, 0
	}

	data, err := os.ReadFile("/sys/fs/cgroup/memory.current")
	if err == nil {
		mem, _ := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
		memBytes = mem
	}

	return 0, memBytes
}
