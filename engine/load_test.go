package engine

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/api"
	"github.com/yaoapp/gou/application/yaz"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/pack"
)

func TestLoad(t *testing.T) {
	defer Unload()
	err := Load(config.Conf, LoadOption{})
	assert.Nil(t, err)
	assert.Greater(t, len(api.APIs), 0)
}

func TestReload(t *testing.T) {
	defer Unload()
	err := Load(config.Conf, LoadOption{})
	assert.Nil(t, err)

	Reload(config.Conf, LoadOption{})
	assert.Nil(t, err)
	assert.Greater(t, len(api.APIs), 0)
}

func TestLoadYaz(t *testing.T) {

	defer Unload()

	// package yaz
	file, err := yaz.Pack(config.Conf.Root, pack.Cipher)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file)

	cfg := config.Conf
	cfg.AppSource = file
	err = Load(cfg, LoadOption{})
	if err != nil {
		t.Fatal(err)
	}
	assert.Greater(t, len(api.APIs), 0)

}

func TestReoadYaz(t *testing.T) {

	defer Unload()

	// package yaz
	file, err := yaz.Pack(config.Conf.Root, pack.Cipher)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file)

	cfg := config.Conf
	cfg.AppSource = file
	err = Load(cfg, LoadOption{})
	if err != nil {
		t.Fatal(err)
	}
	assert.Greater(t, len(api.APIs), 0)

	Reload(cfg, LoadOption{})
	assert.Nil(t, err)
	assert.Greater(t, len(api.APIs), 0)
}
