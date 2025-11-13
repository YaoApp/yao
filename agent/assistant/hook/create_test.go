package hook_test

import (
	"testing"

	"github.com/yaoapp/yao/agent/testutils"
)

// TestCreate test the create hook
func TestCreate(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)
}
