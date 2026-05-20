//go:build unit

package opencode_test

import (
	"strings"
	"testing"

	agentContext "github.com/yaoapp/yao/agent/context"
	opencode "github.com/yaoapp/yao/agent/sandbox/v2/opencode"
)

func TestChatIDToSessionID(t *testing.T) {
	id1 := opencode.ChatIDToSessionID("assistant-1", "chat-1")
	id2 := opencode.ChatIDToSessionID("assistant-1", "chat-1")
	id3 := opencode.ChatIDToSessionID("assistant-1", "chat-2")

	if id1 != id2 {
		t.Error("same inputs should produce same session ID")
	}
	if id1 == id3 {
		t.Error("different chatIDs should produce different session IDs")
	}
	if id1 == "" {
		t.Error("session ID should not be empty")
	}
}

func TestSanitizeSessionName(t *testing.T) {
	cases := []struct {
		input, want string
	}{
		{"simple-chat", "yao-oc-simple-chat"},
		{"chat with spaces", "yao-oc-chat_with_spaces"},
		{"chat/with/slashes", "yao-oc-chat_with_slashes"},
		{"chat@special#chars", "yao-oc-chat_special_chars"},
	}
	for _, tc := range cases {
		got := opencode.SanitizeSessionName(tc.input)
		if got != tc.want {
			t.Errorf("SanitizeSessionName(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestLastUserText(t *testing.T) {
	cases := []struct {
		name     string
		messages []agentContext.Message
		want     string
	}{
		{
			name:     "empty",
			messages: nil,
			want:     "",
		},
		{
			name: "single user message",
			messages: []agentContext.Message{
				{Role: "user", Content: "hello"},
			},
			want: "hello",
		},
		{
			name: "last user wins",
			messages: []agentContext.Message{
				{Role: "user", Content: "first"},
				{Role: "assistant", Content: "reply"},
				{Role: "user", Content: "second"},
			},
			want: "second",
		},
		{
			name: "no user messages",
			messages: []agentContext.Message{
				{Role: "assistant", Content: "only assistant"},
			},
			want: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := opencode.LastUserText(tc.messages)
			if got != tc.want {
				t.Errorf("LastUserText() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestBuildStdinMessage(t *testing.T) {
	msgs := []agentContext.Message{
		{Role: "user", Content: "把这个会议纪要的关键内容提取出来"},
	}

	t.Run("no attachments", func(t *testing.T) {
		got := opencode.BuildStdinMessage(msgs, nil)
		if got != "把这个会议纪要的关键内容提取出来" {
			t.Errorf("unexpected: %q", got)
		}
	})

	t.Run("with attachments", func(t *testing.T) {
		got := opencode.BuildStdinMessage(msgs, []string{"/workspace/.attachments/abc/test.txt"})
		if !strings.Contains(got, "/workspace/.attachments/abc/test.txt") {
			t.Error("should contain attachment path")
		}
		if !strings.Contains(got, "把这个会议纪要的关键内容提取出来") {
			t.Error("should contain user message")
		}
	})

	t.Run("empty message", func(t *testing.T) {
		got := opencode.BuildStdinMessage(nil, []string{"/workspace/file.txt"})
		if !strings.Contains(got, "/workspace/file.txt") {
			t.Error("should contain attachment path even without message")
		}
	})
}

func TestBuildSandboxEnvPrompt(t *testing.T) {
	p := opencode.NewPosixBase("linux", "", "bash")
	prompt := opencode.BuildSandboxEnvPrompt(p, "/workspace")
	if prompt == "" {
		t.Error("prompt should not be empty")
	}
	if !strings.Contains(prompt, "/workspace") {
		t.Error("prompt should mention workspace path")
	}
	if !strings.Contains(prompt, "linux") {
		t.Error("prompt should mention OS")
	}
}

func TestGetProviderPrefix(t *testing.T) {
	if p := opencode.GetProviderPrefix(nil); p != "openai" {
		t.Errorf("nil connector should give openai, got %s", p)
	}
}
