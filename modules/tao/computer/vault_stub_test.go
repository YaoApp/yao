package computer_test

import (
	"context"
	"errors"
	"testing"

	taocomputer "github.com/yaoapp/yao/modules/tao/computer"
)

func TestVaultSSHKeyResolverStub(t *testing.T) {
	if _, err := (taocomputer.VaultSSHKeyResolver{}).ResolveSSHKey(context.Background(), "vault://user/key"); !errors.Is(err, taocomputer.ErrVaultSSHKeyUnavailable) {
		t.Fatalf("stub error=%v", err)
	}
	resolver := taocomputer.VaultSSHKeyResolver{Get: func(context.Context, string) (string, error) { return "private", nil }}
	key, err := resolver.ResolveSSHKey(context.Background(), "vault://user/key")
	if err != nil || key != "private" {
		t.Fatalf("key=%q err=%v", key, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := resolver.ResolveSSHKey(ctx, "vault://user/key"); err == nil {
		t.Fatal("expected context error")
	}
}
