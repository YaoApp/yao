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

	root := config.Conf.DataRoot

	info, err := os.Stat(filepath.Join(root, "test.file"))
	assert.Nil(t, err)
	assert.Equal(t, int64(2), info.Size())

	err = fs.Remove(data, "test.file")
	if err != nil {
		t.Fatal(err)
	}
}

func TestDSL(t *testing.T) {
	Load(config.Conf)
	dsl := fs.MustRootGet("dsl")
	name := filepath.Join("models", "test.mod.json")
	_, err := fs.WriteFile(dsl, name, []byte(`{"foo": "bar", "hello":{ "int": 1, "float": 0.618}}`), 0644)
	assert.Nil(t, err)

	info, err := os.Stat(filepath.Join(config.Conf.Root, name))
	assert.Nil(t, err)
	assert.Equal(t, int64(69), info.Size())
	dsl.Remove(name)
	if err != nil {
		t.Fatal(err)
	}

	_, err = fs.Get("dsl")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "dsl does not registered")

}

func TestScirpt(t *testing.T) {
	Load(config.Conf)
	script := fs.MustRootGet("script")
	name := "test.js"
	_, err := fs.WriteFile(script, name, []byte(`console.log("hello")`), 0644)
	assert.Nil(t, err)

	info, err := os.Stat(filepath.Join(config.Conf.Root, "scripts", name))
	assert.Nil(t, err)
	assert.Equal(t, int64(20), info.Size())
	script.Remove(name)
	if err != nil {
		t.Fatal(err)
	}

	_, err = fs.Get("script")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "script does not registered")
}
