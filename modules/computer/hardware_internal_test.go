package computer

import (
	"testing"
	"time"

	"github.com/yaoapp/yao/modules/computer/types"
)

func TestComputerFromRowHardwareSnapshot(t *testing.T) {
	row := map[string]interface{}{
		"id": "cmp-1", "node_id": "node-1", "name": "mac", "status": "online",
		"os": "darwin", "arch": "arm64", "cpu_cores": 12,
		"memory_mb": uint64(32768), "disk_free_mb": uint64(512000),
		"tai_version":        "2.0.0",
		"gpus":               `[{"name":"M3 Ultra","vram_mb":49152,"driver":"1.0"}]`,
		"capabilities":       `["session","sandbox:native"]`,
		"container_runtimes": `[{"name":"docker","version":"27","socket":"/var/run/docker.sock"}]`,
		"networks":           `[{"id":"en0","name":"en0","kind":"lan","family":"ipv4","address":"192.168.3.110","prefix_length":24,"primary":true}]`,
	}
	c := computerFromRow(row)
	if c.Hardware == nil || c.Hardware.OS != "darwin" || c.Hardware.Arch != "arm64" {
		t.Fatalf("hardware=%+v", c.Hardware)
	}
	if c.Hardware.CPUCores != 12 || c.Hardware.MemoryMB != 32768 || c.Hardware.DiskFreeMB != 512000 {
		t.Fatalf("hardware numbers=%+v", c.Hardware)
	}
	if len(c.Hardware.GPUs) != 1 || c.Hardware.GPUs[0].VRAMMB != 49152 {
		t.Fatalf("gpus=%+v", c.Hardware.GPUs)
	}
	if len(c.Hardware.ContainerRuntimes) != 1 || c.Hardware.ContainerRuntimes[0].Socket != "/var/run/docker.sock" {
		t.Fatalf("runtimes=%+v", c.Hardware.ContainerRuntimes)
	}
	if c.IP != "192.168.3.110" || len(c.Networks) != 1 || !c.Networks[0].Primary {
		t.Fatalf("network=%s %+v", c.IP, c.Networks)
	}
}

func TestComputerFromRowWithoutSnapshot(t *testing.T) {
	c := computerFromRow(map[string]interface{}{"id": "cmp-2", "status": "pending"})
	if c.Hardware != nil || c.IP != "" || len(c.Networks) != 0 {
		t.Fatalf("empty snapshot must stay omitted: %+v", c)
	}
}

func TestSnapshotValuesNil(t *testing.T) {
	if snapshotValues(nil) != nil {
		t.Fatal("nil snapshot should return nil values")
	}
}

func TestSnapshotValuesNormalizesEmptyCollections(t *testing.T) {
	values := snapshotValues(&types.ComputerSnapshot{
		OS: "darwin", CPUCores: 12, LastSeenAt: time.Unix(100, 0),
	})
	for _, key := range []string{"gpus", "capabilities", "container_runtimes", "networks"} {
		switch value := values[key].(type) {
		case []types.GPU:
			if value == nil {
				t.Fatalf("%s should be non-nil empty", key)
			}
		case []string:
			if value == nil {
				t.Fatalf("%s should be non-nil empty", key)
			}
		case []types.ContainerRuntime:
			if value == nil {
				t.Fatalf("%s should be non-nil empty", key)
			}
		case []types.Network:
			if value == nil {
				t.Fatalf("%s should be non-nil empty", key)
			}
		default:
			t.Fatalf("unexpected %s type %T", key, value)
		}
	}
	if values["last_seen_at"] != time.Unix(100, 0) {
		t.Fatalf("last_seen_at=%v", values["last_seen_at"])
	}
}

func TestDecodeJSONAndNumberHelpers(t *testing.T) {
	var networks []types.Network
	if !decodeJSON([]byte(`[{"address":"1.2.3.4","kind":"lan","family":"ipv4"}]`), &networks) || len(networks) != 1 {
		t.Fatalf("[]byte json: %+v", networks)
	}
	networks = nil
	if !decodeJSON([]interface{}{map[string]interface{}{"address": "1.2.3.4", "kind": "lan", "family": "ipv4"}}, &networks) || len(networks) != 1 {
		t.Fatalf("structured json: %+v", networks)
	}
	if decodeJSON("not-json", &networks) {
		t.Fatal("bad json should fail")
	}
	if decodeJSON(nil, &networks) {
		t.Fatal("nil json should fail")
	}
	if decodeJSON("", &networks) {
		t.Fatal("empty string json should fail")
	}
	if decodeJSON(make(chan int), &networks) {
		t.Fatal("unsupported json type should fail")
	}
	cases := []struct {
		value interface{}
		want  uint64
		ok    bool
	}{
		{int(2), 2, true}, {int64(3), 3, true}, {uint32(4), 4, true},
		{float64(5), 5, true}, {"6", 6, true}, {-1, 0, false}, {nil, 0, false},
		{int8(2), 2, true}, {int16(3), 3, true}, {int32(4), 4, true},
		{uint(5), 5, true}, {uint8(6), 6, true}, {uint16(7), 7, true}, {uint64(8), 8, true},
		{float32(9), 9, true}, {"bad", 0, false}, {true, 0, false},
	}
	for _, tc := range cases {
		got, ok := uint64Value(tc.value)
		if got != tc.want || ok != tc.ok {
			t.Fatalf("uint64Value(%T)=%d,%v want %d,%v", tc.value, got, ok, tc.want, tc.ok)
		}
	}
	if got, ok := uint32Value(uint64(7)); !ok || got != 7 {
		t.Fatalf("uint32Value=%d,%v", got, ok)
	}
}

func TestComputerFromRowPurchaseSnapshot(t *testing.T) {
	c := computerFromRow(map[string]interface{}{
		"id": "cmp-cloud", "status": "online",
		"purchase": `{"resource":"cvm-2c2g","region":"gz-cn","image":"tao-base-1.0","billing_unit":"HOUR","period_months":null,"price_credits":480,"price_total_credits":null,"expires_at":null,"purchased_at":"2026-11-26T10:00:00Z"}`,
	})
	if c.Purchase == nil || c.Purchase.Resource != "cvm-2c2g" || c.Purchase.Image != "tao-base-1.0" || c.Purchase.PriceCredits != 480 {
		t.Fatalf("purchase=%+v", c.Purchase)
	}
	if c.Purchase.PeriodMonths != nil {
		t.Fatalf("period should be nil: %+v", c.Purchase.PeriodMonths)
	}
}
