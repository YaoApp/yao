package xfs

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/xiang/config"
)

func TestOSFsMustMkdirAll(t *testing.T) {
	fs := New(config.Conf.Source)
	assert.NotPanics(t, func() {
		fs.MustMkdirAll("/.tmp/unit-test/dir", os.ModePerm)
	})
}
