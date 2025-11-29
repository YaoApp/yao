package types

import (
	"encoding/json"
	"testing"
)

func TestMCPServerConfig_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    MCPServerConfig
		wantErr bool
	}{
		{
			name:  "Simple string",
			input: `"server1"`,
			want: MCPServerConfig{
				ServerID:  "server1",
				Resources: nil,
				Tools:     nil,
			},
			wantErr: false,
		},
		{
			name:  "Tools array only",
			input: `{"server1": ["tool1", "tool2"]}`,
			want: MCPServerConfig{
				ServerID:  "server1",
				Resources: nil,
				Tools:     []string{"tool1", "tool2"},
			},
			wantErr: false,
		},
		{
			name:  "Full config with resources and tools",
			input: `{"server1": {"resources": ["res1", "res2"], "tools": ["tool1", "tool2"]}}`,
			want: MCPServerConfig{
				ServerID:  "server1",
				Resources: []string{"res1", "res2"},
				Tools:     []string{"tool1", "tool2"},
			},
			wantErr: false,
		},
		{
			name:  "Only resources",
			input: `{"server1": {"resources": ["res1"]}}`,
			want: MCPServerConfig{
				ServerID:  "server1",
				Resources: []string{"res1"},
				Tools:     nil,
			},
			wantErr: false,
		},
		{
			name:  "Only tools",
			input: `{"server1": {"tools": ["tool1"]}}`,
			want: MCPServerConfig{
				ServerID:  "server1",
				Resources: nil,
				Tools:     []string{"tool1"},
			},
			wantErr: false,
		},
		{
			name:  "Standard object format",
			input: `{"server_id": "server1", "resources": ["res1"], "tools": ["tool1"]}`,
			want: MCPServerConfig{
				ServerID:  "server1",
				Resources: []string{"res1"},
				Tools:     []string{"tool1"},
			},
			wantErr: false,
		},
		{
			name:  "Standard object format - no resources/tools",
			input: `{"server_id": "server1"}`,
			want: MCPServerConfig{
				ServerID:  "server1",
				Resources: nil,
				Tools:     nil,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got MCPServerConfig
			err := json.Unmarshal([]byte(tt.input), &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if got.ServerID != tt.want.ServerID {
					t.Errorf("ServerID = %v, want %v", got.ServerID, tt.want.ServerID)
				}
				if !stringSlicesEqual(got.Resources, tt.want.Resources) {
					t.Errorf("Resources = %v, want %v", got.Resources, tt.want.Resources)
				}
				if !stringSlicesEqual(got.Tools, tt.want.Tools) {
					t.Errorf("Tools = %v, want %v", got.Tools, tt.want.Tools)
				}
			}
		})
	}
}

func TestMCPServers_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []MCPServerConfig
		wantErr bool
	}{
		{
			name:  "Simple string array",
			input: `{"servers": ["server1", "server2", "server3"]}`,
			want: []MCPServerConfig{
				{ServerID: "server1"},
				{ServerID: "server2"},
				{ServerID: "server3"},
			},
			wantErr: false,
		},
		{
			name:  "Mixed formats",
			input: `{"servers": ["server1", {"server2": ["tool1", "tool2"]}, {"server3": {"resources": ["res1"], "tools": ["tool3"]}}]}`,
			want: []MCPServerConfig{
				{ServerID: "server1"},
				{ServerID: "server2", Tools: []string{"tool1", "tool2"}},
				{ServerID: "server3", Resources: []string{"res1"}, Tools: []string{"tool3"}},
			},
			wantErr: false,
		},
		{
			name:    "Empty servers",
			input:   `{"servers": []}`,
			want:    []MCPServerConfig{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got MCPServers
			err := json.Unmarshal([]byte(tt.input), &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(got.Servers) != len(tt.want) {
					t.Errorf("got %d servers, want %d", len(got.Servers), len(tt.want))
					return
				}

				for i := range got.Servers {
					if got.Servers[i].ServerID != tt.want[i].ServerID {
						t.Errorf("Server[%d].ServerID = %v, want %v", i, got.Servers[i].ServerID, tt.want[i].ServerID)
					}
					if !stringSlicesEqual(got.Servers[i].Resources, tt.want[i].Resources) {
						t.Errorf("Server[%d].Resources = %v, want %v", i, got.Servers[i].Resources, tt.want[i].Resources)
					}
					if !stringSlicesEqual(got.Servers[i].Tools, tt.want[i].Tools) {
						t.Errorf("Server[%d].Tools = %v, want %v", i, got.Servers[i].Tools, tt.want[i].Tools)
					}
				}
			}
		})
	}
}

func TestMCPServerConfig_MarshalJSON(t *testing.T) {
	tests := []struct {
		name   string
		config MCPServerConfig
		want   string
	}{
		{
			name: "Only ServerID - should be simple string",
			config: MCPServerConfig{
				ServerID: "server1",
			},
			want: `"server1"`,
		},
		{
			name: "With Tools - should be object",
			config: MCPServerConfig{
				ServerID: "server1",
				Tools:    []string{"tool1", "tool2"},
			},
			want: `{"server_id":"server1","tools":["tool1","tool2"]}`,
		},
		{
			name: "With Resources - should be object",
			config: MCPServerConfig{
				ServerID:  "server1",
				Resources: []string{"res1"},
			},
			want: `{"server_id":"server1","resources":["res1"]}`,
		},
		{
			name: "With Both - should be object",
			config: MCPServerConfig{
				ServerID:  "server1",
				Resources: []string{"res1"},
				Tools:     []string{"tool1"},
			},
			want: `{"server_id":"server1","resources":["res1"],"tools":["tool1"]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.config)
			if err != nil {
				t.Errorf("MarshalJSON() error = %v", err)
				return
			}
			if string(got) != tt.want {
				t.Errorf("MarshalJSON() = %s, want %s", string(got), tt.want)
			}
		})
	}
}

func TestMCPServerConfig_RoundTrip(t *testing.T) {
	tests := []struct {
		name   string
		config MCPServerConfig
	}{
		{
			name: "Simple ServerID",
			config: MCPServerConfig{
				ServerID: "server1",
			},
		},
		{
			name: "With Tools",
			config: MCPServerConfig{
				ServerID: "server2",
				Tools:    []string{"tool1", "tool2"},
			},
		},
		{
			name: "With Resources and Tools",
			config: MCPServerConfig{
				ServerID:  "server3",
				Resources: []string{"res1", "res2"},
				Tools:     []string{"tool3", "tool4"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal
			data, err := json.Marshal(tt.config)
			if err != nil {
				t.Fatalf("Marshal error = %v", err)
			}

			// Unmarshal
			var got MCPServerConfig
			err = json.Unmarshal(data, &got)
			if err != nil {
				t.Fatalf("Unmarshal error = %v", err)
			}

			// Compare
			if got.ServerID != tt.config.ServerID {
				t.Errorf("ServerID = %v, want %v", got.ServerID, tt.config.ServerID)
			}
			if !stringSlicesEqual(got.Resources, tt.config.Resources) {
				t.Errorf("Resources = %v, want %v", got.Resources, tt.config.Resources)
			}
			if !stringSlicesEqual(got.Tools, tt.config.Tools) {
				t.Errorf("Tools = %v, want %v", got.Tools, tt.config.Tools)
			}
		})
	}
}

// Helper function to compare string slices (nil-safe)
func stringSlicesEqual(a, b []string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
