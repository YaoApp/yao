package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/openapi/oauth"
)

type fakeValidator struct{ err error }

func (f fakeValidator) ValidateNodeToken(ctx context.Context, raw string) (*oauth.NodeInfo, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &oauth.NodeInfo{NodeID: "node-test"}, nil
}

func TestProvider(t *testing.T) {
	p := New(fakeValidator{})
	info, err := p.Validate(context.Background(), "raw")
	require.NoError(t, err)
	require.Equal(t, "node-test", info.NodeID)

	_, err = New(fakeValidator{err: errors.New("bad")}).Validate(context.Background(), "raw")
	require.Error(t, err)
	_, err = (&Provider{}).Validate(context.Background(), "raw")
	require.Error(t, err)
}
