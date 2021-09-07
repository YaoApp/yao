package main

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/kun/any"
)

func TestNewConfig(t *testing.T) {
	cfg := NewConfig()
	var vBool = func(name string) bool {
		if name == "true" || name == "1" {
			return true
		}
		return false
	}

	assert.Equal(t, cfg.Mode, os.Getenv("XIANG_MODE"))
	assert.Equal(t, cfg.Source, os.Getenv("XIANG_SOURCE"))
	assert.Equal(t, cfg.Root, os.Getenv("XIANG_ROOT"))

	assert.Equal(t, cfg.Service.Debug, vBool(os.Getenv("XIANG_SERVICE_DEBUG")))
	assert.Equal(t, strings.Join(cfg.Service.Allow, "|"), os.Getenv("XIANG_SERVICE_ALLOW"))
	assert.Equal(t, cfg.Service.Host, os.Getenv("XIANG_SERVICE_HOST"))
	assert.Equal(t, cfg.Service.Port, any.Of(os.Getenv("XIANG_SERVICE_PORT")).CInt())

	assert.Equal(t, cfg.Database.Debug, vBool(os.Getenv("XIANG_DB_DEBUG")))
	assert.Equal(t, strings.Join(cfg.Database.Primary, "|"), os.Getenv("XIANG_DB_PRIMARY"))
	assert.Equal(t, strings.Join(cfg.Database.Secondary, "|"), os.Getenv("XIANG_DB_SECONDARY"))
	assert.Equal(t, cfg.Database.AESKey, os.Getenv("XIANG_DB_AESKEY"))

	assert.Equal(t, cfg.JWT.Debug, vBool(os.Getenv("XIANG_JWT_DEBUG")))
	assert.Equal(t, cfg.JWT.Secret, os.Getenv("XIANG_JWT_SECRET"))

	assert.Equal(t, cfg.Storage.Debug, vBool(os.Getenv("XIANG_STOR_DEBUG")))
	assert.Equal(t, cfg.Storage.Path, os.Getenv("XIANG_STOR_PATH"))

	assert.Equal(t, cfg.Log.Access, os.Getenv("XIANG_LOG_ACCESS"))
	assert.Equal(t, cfg.Log.Error, os.Getenv("XIANG_LOG_ERROR"))
	assert.Equal(t, cfg.Log.DB, os.Getenv("XIANG_LOG_DB"))
	assert.Equal(t, cfg.Log.Plugin, os.Getenv("XIANG_LOG_PLUGIN"))

}
