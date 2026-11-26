package tao

import (
	"context"
	"errors"
	"fmt"

	computertypes "github.com/yaoapp/yao/modules/computer/types"
	taocomputer "github.com/yaoapp/yao/modules/tao/computer"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/user"
)

// CloudComputerReleaser deletes the upstream Tao instance for a cloud
// Computer before Yao removes the local record.
type CloudComputerReleaser struct {
	client *taocomputer.Client
}

// NewCloudComputerReleaser creates a Tao release adapter from the environment
// configuration (TAO_ENDPOINT / TAO_TIMEOUT) at composition time.
func NewCloudComputerReleaser() *CloudComputerReleaser {
	return &CloudComputerReleaser{client: taocomputer.NewClient("")}
}

// Release resolves the billing user's Tao key and deletes the instance.
// Tao DELETE is idempotent: accepted or already absent both count as success.
func (r *CloudComputerReleaser) Release(ctx context.Context, item *computertypes.Computer) error {
	if item == nil {
		return errors.New("cloud computer is required")
	}
	if item.InstanceID == "" {
		return errors.New("cloud computer has no instance_id")
	}
	key, err := user.ResolveTaoBillingKey(ctx, &oauthtypes.AuthorizedInfo{
		UserID: item.UserID,
		TeamID: item.TeamID,
	})
	if err != nil {
		return err
	}
	if err := r.client.Delete(ctx, key, item.InstanceID); err != nil {
		var apiErr *taocomputer.APIError
		if errors.As(err, &apiErr) && apiErr.Status == 404 {
			return nil
		}
		return fmt.Errorf("delete Tao instance %s: %w", item.InstanceID, err)
	}
	return nil
}
