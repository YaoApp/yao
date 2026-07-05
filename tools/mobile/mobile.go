package mobile

import (
	"context"
	"fmt"

	"github.com/yaoapp/gou/process"
	tai "github.com/yaoapp/yao/tai"
	hepb "github.com/yaoapp/yao/tai/hostexec/pb"
	"github.com/yaoapp/yao/tai/registry"
	"github.com/yaoapp/yao/tai/types"
)

func listAndroidNodes() []types.NodeMeta {
	reg := registry.Global()
	if reg == nil {
		return nil
	}
	all := reg.List()
	var result []types.NodeMeta
	for _, n := range all {
		if n.System.OS == "android" && n.Status == "online" {
			result = append(result, n)
		}
	}
	return result
}

func getAndroidNode(deviceID string) (*types.NodeMeta, error) {
	meta, ok := tai.GetNodeMeta(deviceID)
	if !ok {
		return nil, fmt.Errorf("device %q not found", deviceID)
	}
	if meta.Status != "online" {
		return nil, fmt.Errorf("device %q is offline", deviceID)
	}
	if meta.System.OS != "android" {
		return nil, fmt.Errorf("device %q is not an Android device (os=%s)", deviceID, meta.System.OS)
	}
	return meta, nil
}

func getHostExec(deviceID string) (hepb.HostExecClient, func(), error) {
	res, ok := tai.GetResources(deviceID)
	if !ok {
		return nil, nil, fmt.Errorf("device %q: no connection resources", deviceID)
	}
	if res.HostExec == nil {
		return nil, nil, fmt.Errorf("device %q: host_exec not enabled", deviceID)
	}
	return res.HostExec, func() {}, nil
}

func execOnDevice(ctx context.Context, deviceID, command string) (string, string, int32, error) {
	he, cleanup, err := getHostExec(deviceID)
	if err != nil {
		return "", "", -1, err
	}
	defer cleanup()

	resp, err := he.Exec(ctx, &hepb.ExecRequest{
		Command:   "sh",
		Args:      []string{"-c", command},
		TimeoutMs: 60000,
	})
	if err != nil {
		return "", "", -1, fmt.Errorf("exec rpc: %w", err)
	}

	stdout := string(resp.Stdout)
	stderr := string(resp.Stderr)
	if resp.Error != "" {
		return stdout, stderr, resp.ExitCode, fmt.Errorf("%s", resp.Error)
	}
	return stdout, stderr, resp.ExitCode, nil
}

func extractDeviceID(proc *process.Process, idx int) (string, error) {
	id := proc.ArgsString(idx)
	if id == "" {
		return "", fmt.Errorf("device_id is required")
	}
	return id, nil
}
