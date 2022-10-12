package xfs

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// DEPRECATED

func TestOSFsMustMkdirAll(t *testing.T) {
	// fs := New(config.Conf.Source)
	fs := New(os.Getenv("YAO_DEV"))
	assert.NotPanics(t, func() {
		fs.MustMkdirAll("/.tmp/unit-test/dir", os.ModePerm)
	})
}
