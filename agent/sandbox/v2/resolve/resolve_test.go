package resolve

import "testing"

func TestResolveMode(t *testing.T) {
	tests := []struct {
		name   string
		node   NodeCandidate
		runner string
		image  string
		want   string
	}{
		{
			name:   "local yaocode",
			node:   NodeCandidate{IsLocal: true, CanBox: true, CanHost: true},
			runner: "yaocode",
			want:   "local",
		},
		{
			name:   "local non-yaocode runner with image → box",
			node:   NodeCandidate{IsLocal: true, CanBox: true, CanHost: true},
			runner: "tai",
			image:  "alpine:latest",
			want:   "box",
		},
		{
			name:  "remote canBox with image → box",
			node:  NodeCandidate{CanBox: true, CanHost: true},
			image: "ubuntu:22.04",
			want:  "box",
		},
		{
			name: "remote canHost only → host",
			node: NodeCandidate{CanHost: true},
			want: "host",
		},
		{
			name: "remote canBox no image → host preferred when available",
			node: NodeCandidate{CanBox: true, CanHost: true},
			want: "host",
		},
		{
			name: "remote canBox no image no host → box",
			node: NodeCandidate{CanBox: true},
			want: "box",
		},
		{
			name: "no capabilities → empty",
			node: NodeCandidate{},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveMode(&tt.node, tt.runner, tt.image)
			if got != tt.want {
				t.Errorf("ResolveMode() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildIdentifier(t *testing.T) {
	tests := []struct {
		name        string
		lifecycle   string
		ownerID     string
		chatID      string
		assistantID string
		workspaceID string
		want        string
	}{
		{
			name:      "oneshot → empty",
			lifecycle: "oneshot",
			ownerID:   "u1", chatID: "c1", assistantID: "a1", workspaceID: "w1",
			want: "",
		},
		{
			name:      "session",
			lifecycle: "session",
			ownerID:   "team-x", chatID: "chat-42", assistantID: "asst-7",
			want: "team-x-asst-7-chat-42",
		},
		{
			name:      "longrunning",
			lifecycle: "longrunning",
			ownerID:   "u1", assistantID: "a1", workspaceID: "ws-abc",
			want: "u1-a1.ws-abc",
		},
		{
			name:      "persistent",
			lifecycle: "persistent",
			ownerID:   "u1", assistantID: "a1", workspaceID: "ws-abc",
			want: "u1-a1.ws-abc",
		},
		{
			name:      "unknown lifecycle → empty",
			lifecycle: "unknown",
			ownerID:   "u1", chatID: "c1", assistantID: "a1", workspaceID: "w1",
			want: "",
		},
		{
			name:      "empty lifecycle → empty",
			lifecycle: "",
			ownerID:   "u1",
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildIdentifier(tt.lifecycle, tt.ownerID, tt.chatID, tt.assistantID, tt.workspaceID)
			if got != tt.want {
				t.Errorf("BuildIdentifier() = %q, want %q", got, tt.want)
			}
		})
	}
}
