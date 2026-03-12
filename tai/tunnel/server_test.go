package tunnel

import (
	"net/http"
	"testing"

	"github.com/yaoapp/yao/tai/tunnel/taipb"
	"github.com/yaoapp/yao/tai/types"
)

func TestExtractBearer(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   string
	}{
		{"valid", "Bearer abc123", "abc123"},
		{"lowercase", "bearer xyz", "xyz"},
		{"empty", "", ""},
		{"no_scheme", "abc123", ""},
		{"only_bearer", "Bearer ", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &http.Request{Header: http.Header{}}
			if tt.header != "" {
				r.Header.Set("Authorization", tt.header)
			}
			got := extractBearer(r)
			if got != tt.want {
				t.Errorf("extractBearer(%q) = %q, want %q", tt.header, got, tt.want)
			}
		})
	}
}

func TestPortsFromMap(t *testing.T) {
	m := map[string]int{"grpc": 19100, "http": 8099, "vnc": 16080, "docker": 12375, "k8s": 16443}
	p := portsFromMap(m)
	if p.GRPC != 19100 || p.HTTP != 8099 || p.VNC != 16080 || p.Docker != 12375 || p.K8s != 16443 {
		t.Errorf("portsFromMap got %+v", p)
	}
}

func TestPortsFromMap_Empty(t *testing.T) {
	p := portsFromMap(nil)
	if p.GRPC != 0 || p.HTTP != 0 {
		t.Errorf("portsFromMap(nil) got %+v", p)
	}
}

func TestCapsFromMap(t *testing.T) {
	m := map[string]bool{"docker": true, "k8s": false, "host_exec": true}
	c := capsFromMap(m)
	if !c.Docker || c.K8s || !c.HostExec {
		t.Errorf("capsFromMap got %+v", c)
	}
}

func TestCapsFromMap_Empty(t *testing.T) {
	c := capsFromMap(nil)
	if c.Docker || c.K8s || c.HostExec {
		t.Errorf("capsFromMap(nil) got %+v", c)
	}
}

func TestPortsFromProto(t *testing.T) {
	pp := &taipb.Ports{Grpc: 19100, Http: 8099, Vnc: 16080, Docker: 12375, K8S: 16443}
	p := portsFromProto(pp)
	if p.GRPC != 19100 || p.HTTP != 8099 || p.VNC != 16080 || p.Docker != 12375 || p.K8s != 16443 {
		t.Errorf("portsFromProto got %+v", p)
	}
}

func TestPortsFromProto_Nil(t *testing.T) {
	p := portsFromProto(nil)
	if p != (types.Ports{}) {
		t.Errorf("portsFromProto(nil) = %+v", p)
	}
}

func TestCapsFromProto(t *testing.T) {
	cp := &taipb.Capabilities{Docker: true, K8S: false, HostExec: true}
	c := capsFromProto(cp)
	if !c.Docker || c.K8s || !c.HostExec {
		t.Errorf("capsFromProto got %+v", c)
	}
}

func TestCapsFromProto_Nil(t *testing.T) {
	c := capsFromProto(nil)
	if c != (types.Capabilities{}) {
		t.Errorf("capsFromProto(nil) = %+v", c)
	}
}

func TestSystemFromProto(t *testing.T) {
	sp := &taipb.SystemInfo{Os: "linux", Arch: "amd64", Hostname: "host1", Shell: "bash"}
	s := systemFromProto(sp)
	if s.OS != "linux" || s.Arch != "amd64" || s.Hostname != "host1" || s.Shell != "bash" {
		t.Errorf("systemFromProto got %+v", s)
	}
}

func TestSystemFromProto_Nil(t *testing.T) {
	s := systemFromProto(nil)
	if s != (types.SystemInfo{}) {
		t.Errorf("systemFromProto(nil) = %+v", s)
	}
}

func TestAuthenticateBearerDefault_NoOAuth(t *testing.T) {
	_, err := authenticateBearerDefault("some-token")
	if err == nil {
		t.Fatal("expected error when oauth service is nil")
	}
}

func TestAuthenticateBearerFunc_IsDefault(t *testing.T) {
	if authenticateBearerFunc == nil {
		t.Fatal("authenticateBearerFunc should be set")
	}
}
