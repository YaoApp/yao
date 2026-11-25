package tai

import "testing"

func TestNodeSnapshotCarriesHardwareAndPrimaryIP(t *testing.T) {
	snapshot := nodeSnapshot(Node{
		Version: "2.0.0", OS: "darwin", Arch: "arm64",
		CPUCores: 12, MemoryMB: 32768, DiskFreeMB: 512000,
		GPUs:              []GPU{{Name: "M3 Ultra", VRAMMB: 49152, Driver: "1.0"}},
		Capabilities:      []string{"session"},
		ContainerRuntimes: []Runtime{{Name: "docker", Version: "27", Socket: "/var/run/docker.sock"}},
		Networks: []Network{
			{ID: "lo0", Name: "lo0", Kind: "loopback", Family: "ipv4", Address: "127.0.0.1", PrefixLength: 8},
			{ID: "en0", Name: "en0", Kind: "lan", Family: "ipv4", Address: "192.168.3.110", PrefixLength: 24, Primary: true},
		},
	})
	if snapshot.OS != "darwin" || snapshot.CPUCores != 12 || snapshot.MemoryMB != 32768 || snapshot.DiskFreeMB != 512000 {
		t.Fatalf("hardware=%+v", snapshot)
	}
	if len(snapshot.GPUs) != 1 || snapshot.GPUs[0].VRAMMB != 49152 {
		t.Fatalf("gpus=%+v", snapshot.GPUs)
	}
	if snapshot.IP != "192.168.3.110" || len(snapshot.Networks) != 2 || snapshot.Networks[1].Kind != "lan" {
		t.Fatalf("networks ip=%s list=%+v", snapshot.IP, snapshot.Networks)
	}
	if snapshot.TaiVersion != "2.0.0" || snapshot.LastSeenAt.IsZero() {
		t.Fatalf("version/last-seen=%+v", snapshot)
	}
}
