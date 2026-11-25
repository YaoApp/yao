//go:build e2e

package computers_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/yaoapp/yao/e2e/client"
	"github.com/yaoapp/yao/modules/computer"
	computertypes "github.com/yaoapp/yao/modules/computer/types"
	"github.com/yaoapp/yao/openapi"
	"github.com/yaoapp/yao/openapi/oauth"
	userpkg "github.com/yaoapp/yao/openapi/user"
	taipkg "github.com/yaoapp/yao/tai"
)

type hardwareComputer struct {
	ID       string `json:"id"`
	IP       string `json:"ip"`
	Hardware *struct {
		OS           string   `json:"os"`
		Arch         string   `json:"arch"`
		CPUCores     uint32   `json:"cpu_cores"`
		MemoryMB     uint64   `json:"memory_mb"`
		DiskFreeMB   uint64   `json:"disk_free_mb"`
		TaiVersion   string   `json:"tai_version"`
		Capabilities []string `json:"capabilities"`
		GPUs         []struct {
			Name   string `json:"name"`
			VRAMMB uint64 `json:"vram_mb"`
			Driver string `json:"driver"`
		} `json:"gpus"`
		ContainerRuntimes []struct {
			Name string `json:"name"`
		} `json:"container_runtimes"`
	} `json:"hardware"`
	Networks []struct {
		ID      string `json:"id"`
		Kind    string `json:"kind"`
		Family  string `json:"family"`
		Address string `json:"address"`
		Primary bool   `json:"primary"`
	} `json:"networks"`
}

func TestE2EComputerHardwareSnapshot(t *testing.T) {
	environment := testEnvironment(t)
	service, ok := openapi.Server.OAuth.(*oauth.Service)
	if !ok || service == nil {
		t.Fatal("OAuth service is not available")
	}
	ctx := context.Background()
	provider, err := service.GetUserProvider()
	if err != nil {
		t.Fatalf("get user provider: %v", err)
	}
	admin, err := provider.GetUserByEmail(ctx, "admin@e2e.local")
	if err != nil {
		t.Fatalf("get admin: %v", err)
	}
	login, err := userpkg.LoginByUserID(fmt.Sprint(admin["user_id"]), nil)
	if err != nil {
		t.Fatalf("login admin: %v", err)
	}
	api := client.New(environment.BaseURL)
	api.Token = login.AccessToken

	connResp, connBody, err := api.PostJSON("/v1/computers/connect", map[string]interface{}{
		"name": "Hardware Snapshot E2E", "data_dir": "/tmp/e2e-hardware", "visibility": "private",
	})
	if err != nil || connResp.StatusCode != http.StatusOK {
		t.Fatalf("connect status=%v err=%v body=%s", connResp, err, connBody)
	}
	var created connectResponse
	if err := json.Unmarshal(connBody, &created); err != nil {
		t.Fatalf("decode connect: %v", err)
	}

	computers := computer.New().WithTokens(oauth.OAuth)
	observer := &taipkg.ComputerObserver{Computers: computers}
	if err := observer.NodeRegistered(ctx, taipkg.Node{
		ID: created.NodeID, Version: "2.0.0", OS: "darwin", Arch: "arm64",
		CPUCores: 12, MemoryMB: 32768, DiskFreeMB: 512000,
		Capabilities:      []string{"session", "sandbox:native"},
		ContainerRuntimes: []taipkg.Runtime{{Name: "docker", Version: "27", Socket: "/var/run/docker.sock"}},
		Networks: []taipkg.Network{
			{ID: "lo0", Name: "lo0", Kind: "loopback", Family: "ipv4", Address: "127.0.0.1", PrefixLength: 8},
			{ID: "en0", Name: "en0", Kind: "lan", Family: "ipv4", Address: "192.168.3.110", PrefixLength: 24, Primary: true},
		},
	}); err != nil {
		t.Fatalf("observer registration: %v", err)
	}
	t.Cleanup(func() {
		_ = computers.Delete(context.Background(), created.ComputerID)
	})

	resp, body, err := api.Get("/v1/computers/" + created.ComputerID)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("detail status=%v err=%v body=%s", resp, err, body)
	}
	var detail hardwareComputer
	if err := json.Unmarshal(body, &detail); err != nil {
		t.Fatalf("decode detail: %v", err)
	}
	assertHardwareSnapshot(t, detail, created.ComputerID)

	resp, body, err = api.Get("/v1/computers")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("list status=%v err=%v body=%s", resp, err, body)
	}
	var list struct {
		Data []hardwareComputer `json:"data"`
	}
	if err := json.Unmarshal(body, &list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	found := false
	for _, item := range list.Data {
		if item.ID == created.ComputerID {
			found = true
			assertHardwareSnapshot(t, item, created.ComputerID)
		}
	}
	if !found {
		t.Fatalf("computer %s missing from list: %s", created.ComputerID, body)
	}

	// A second registration replaces the snapshot rather than accumulating stale fields.
	if err := observer.NodeRegistered(ctx, taipkg.Node{
		ID: created.NodeID, Version: "2.0.1", OS: "darwin", Arch: "arm64",
		CPUCores: 16, MemoryMB: 65536, DiskFreeMB: 1024,
		Networks: []taipkg.Network{{ID: "en0", Name: "en0", Kind: "lan", Family: "ipv4", Address: "10.0.0.8", PrefixLength: 24, Primary: true}},
	}); err != nil {
		t.Fatalf("observer re-registration: %v", err)
	}
	resp, body, err = api.Get("/v1/computers/" + created.ComputerID + "/status")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("status after re-register=%v err=%v body=%s", resp, err, body)
	}
	_, body, err = api.Get("/v1/computers/" + created.ComputerID)
	if err != nil {
		t.Fatalf("detail after re-register: %v", err)
	}
	detail = hardwareComputer{}
	if err := json.Unmarshal(body, &detail); err != nil {
		t.Fatalf("decode detail after re-register: %v", err)
	}
	if detail.Hardware == nil || detail.Hardware.TaiVersion != "2.0.1" || detail.Hardware.CPUCores != 16 {
		t.Fatalf("stale hardware after re-register: %+v", detail.Hardware)
	}
	if detail.IP != "10.0.0.8" {
		t.Fatalf("stale ip after re-register: %s", detail.IP)
	}
	if len(detail.Hardware.ContainerRuntimes) != 0 || len(detail.Hardware.Capabilities) != 0 {
		t.Fatalf("stale runtime/capabilities after re-register: %+v", detail.Hardware)
	}
}

func assertHardwareSnapshot(t *testing.T, item hardwareComputer, computerID string) {
	t.Helper()
	if item.ID != computerID || item.Hardware == nil {
		t.Fatalf("missing hardware: %+v", item)
	}
	hw := item.Hardware
	if hw.OS != "darwin" || hw.Arch != "arm64" || hw.CPUCores != 12 ||
		hw.MemoryMB != 32768 || hw.DiskFreeMB != 512000 || hw.TaiVersion != "2.0.0" {
		t.Fatalf("unexpected hardware: %+v", hw)
	}
	if item.IP != "192.168.3.110" || len(item.Networks) != 2 || !item.Networks[1].Primary {
		t.Fatalf("unexpected networks: ip=%s networks=%+v", item.IP, item.Networks)
	}
	if len(hw.GPUs) != 0 || len(hw.ContainerRuntimes) != 1 || hw.ContainerRuntimes[0].Name != "docker" {
		t.Fatalf("unexpected runtimes/gpus: %+v", hw)
	}
	if len(hw.Capabilities) != 2 || hw.Capabilities[0] != "session" {
		t.Fatalf("unexpected capabilities: %+v", hw.Capabilities)
	}
}

var _ computertypes.Computer
