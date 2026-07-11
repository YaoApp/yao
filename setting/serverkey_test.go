package setting_test

import (
	"strings"
	"testing"

	"github.com/yaoapp/yao/setting"
)

func TestIsServerKeyFormat(t *testing.T) {
	tests := []struct {
		token string
		want  bool
	}{
		{"yao-sk:abc123def456", true},
		{"yao-sk:x", true},
		{"yao-sk:", false}, // prefix only, no content
		{"yao-", false},    // API key prefix, not server key
		{"yao-abc", false}, // API key
		{"bearer-token", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := setting.IsServerKeyFormat(tt.token); got != tt.want {
			t.Errorf("IsServerKeyFormat(%q) = %v, want %v", tt.token, got, tt.want)
		}
	}
}

func TestCreateServerKey(t *testing.T) {
	setupRegistry(t)

	plainKey, keyID, err := setting.CreateServerKey("test-node")
	if err != nil {
		t.Fatalf("CreateServerKey: %v", err)
	}

	if !strings.HasPrefix(plainKey, "yao-sk:") {
		t.Errorf("plainKey should start with yao-sk:, got %q", plainKey)
	}
	if !strings.HasPrefix(keyID, "sk-") {
		t.Errorf("keyID should start with sk-, got %q", keyID)
	}
}

func TestValidateServerKey(t *testing.T) {
	setupRegistry(t)

	plainKey, _, err := setting.CreateServerKey("validate-test")
	if err != nil {
		t.Fatalf("CreateServerKey: %v", err)
	}

	// Valid key
	keyID, err := setting.ValidateServerKey(plainKey)
	if err != nil {
		t.Fatalf("ValidateServerKey should succeed: %v", err)
	}
	if keyID == "" {
		t.Error("keyID should not be empty")
	}

	// Invalid key
	_, err = setting.ValidateServerKey("yao-sk:invalid-key")
	if err == nil {
		t.Error("ValidateServerKey should fail for invalid key")
	}

	// Revoked key
	if err := setting.RevokeServerKey(keyID); err != nil {
		t.Fatalf("RevokeServerKey: %v", err)
	}
	_, err = setting.ValidateServerKey(plainKey)
	if err == nil {
		t.Error("ValidateServerKey should fail for revoked key")
	}
}
