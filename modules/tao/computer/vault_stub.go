package computer

import (
	"context"
	"errors"
	"fmt"
)

// ErrVaultSSHKeyUnavailable is returned until the Vault secret backend is wired.
var ErrVaultSSHKeyUnavailable = errors.New("vault ssh key resolver is not configured")

// VaultSSHKeyResolver is the Vault-backed SSH key resolver seam.
// Get may be supplied by a deployment adapter; the default is a stub.
type VaultSSHKeyResolver struct {
	Get func(context.Context, string) (string, error)
}

// ResolveSSHKey resolves a Vault SSH key reference.
func (r VaultSSHKeyResolver) ResolveSSHKey(ctx context.Context, ref string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if r.Get != nil {
		return r.Get(ctx, ref)
	}
	return "", fmt.Errorf("%w: %s", ErrVaultSSHKeyUnavailable, ref)
}

var _ SSHKeyResolver = VaultSSHKeyResolver{}
