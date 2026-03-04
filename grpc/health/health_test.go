package health_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yaoapp/yao/grpc/pb"
	"github.com/yaoapp/yao/grpc/tests/testutils"
)

func TestHealthz_ReturnsOk(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	resp, err := client.Healthz(context.Background(), &pb.Empty{})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "ok", resp.Status)
}
