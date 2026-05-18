package sandbox_test

import (
	"strings"
	"testing"

	sandbox "github.com/yaoapp/yao/sandbox/v2"
)

func TestErrorMessages(t *testing.T) {
	cases := []struct {
		err    error
		substr string
	}{
		{sandbox.ErrNotAvailable, "not available"},
		{sandbox.ErrNotFound, "not found"},
		{sandbox.ErrNodeNotFound, "node not found"},
		{sandbox.ErrNodeMissing, "node ID is required"},
	}
	for _, tc := range cases {
		if !strings.Contains(tc.err.Error(), tc.substr) {
			t.Errorf("error %q does not contain %q", tc.err, tc.substr)
		}
	}
}

func TestErrorsAreDistinct(t *testing.T) {
	errs := []error{
		sandbox.ErrNotAvailable,
		sandbox.ErrNotFound,
		sandbox.ErrNodeNotFound,
		sandbox.ErrNodeMissing,
	}
	for i := 0; i < len(errs); i++ {
		for j := i + 1; j < len(errs); j++ {
			if errs[i] == errs[j] {
				t.Errorf("errors[%d] (%v) == errors[%d] (%v)", i, errs[i], j, errs[j])
			}
		}
	}
}
