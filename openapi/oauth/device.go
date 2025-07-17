package oauth

import (
	"context"

	"github.com/yaoapp/yao/openapi/oauth/types"
)

// DeviceAuthorization initiates the device authorization flow
// This is used for devices with limited input capabilities
func (s *Service) DeviceAuthorization(ctx context.Context, clientID string, scope string) (*types.DeviceAuthorizationResponse, error) {
	// TODO: Implement device authorization flow
	return nil, nil
}
