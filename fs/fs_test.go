package fs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/fs"
	"github.com/yaoapp/yao/config"
)

func TestLoad(t *testing.T) {
	Load(config.Conf)

	data := fs.MustGet("system")
	size, err := fs.WriteFile(data, "test.file", []byte("Hi"), 0644)
	assert.Nil(t, err)
	assert.Equal(t, 2, size)

	root, err := Root(config.Conf)
	assert.Nil(t, err)

	info, err := os.Stat(filepath.Join(root, "test.file"))
	assert.Nil(t, err)
	assert.Equal(t, int64(2), info.Size())

	err = fs.Remove(data, "test.file")
	assert.Nil(t, err)
}
