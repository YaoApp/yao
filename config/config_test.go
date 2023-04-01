package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// func TestNewConfig(t *testing.T) {
// 	cfg := NewConfig()
// 	var vBool = func(name string) bool {
// 		if name == "true" || name == "1" {
// 			return true
// 		}
// 		return false
// 	}

// 	xiangPath := os.Getenv("XIANG_PATH")
// 	if xiangPath == "" {
// 		xiangPath = "bin://xiang"
// 	}

// 	assert.Equal(t, cfg.Mode, os.Getenv("XIANG_MODE"))
// 	assert.Equal(t, cfg.Root, os.Getenv("XIANG_ROOT"))
// 	assert.Equal(t, cfg.Path, xiangPath)

// 	assert.Equal(t, cfg.Service.Debug, vBool(os.Getenv("XIANG_SERVICE_DEBUG")))
// 	assert.Equal(t, strings.Join(cfg.Service.Allow, "|"), os.Getenv("XIANG_SERVICE_ALLOW"))
// 	assert.Equal(t, cfg.Service.Host, os.Getenv("XIANG_SERVICE_HOST"))
// 	assert.Equal(t, cfg.Service.Port, any.Of(os.Getenv("XIANG_SERVICE_PORT")).CInt())

// 	assert.Equal(t, cfg.Database.Debug, vBool(os.Getenv("XIANG_DB_DEBUG")))
// 	assert.Equal(t, strings.Join(cfg.Database.Primary, "|"), os.Getenv("XIANG_DB_PRIMARY"))
// 	assert.Equal(t, strings.Join(cfg.Database.Secondary, "|"), os.Getenv("XIANG_DB_SECONDARY"))
// 	assert.Equal(t, cfg.Database.AESKey, os.Getenv("XIANG_DB_AESKEY"))

// 	assert.Equal(t, cfg.JWT.Secret, os.Getenv("XIANG_JWT_SECRET"))

// 	assert.Equal(t, cfg.Log.Access, os.Getenv("XIANG_LOG_ACCESS"))
// 	assert.Equal(t, cfg.Log.Error, os.Getenv("XIANG_LOG_ERROR"))
// 	assert.Equal(t, cfg.Log.DB, os.Getenv("XIANG_LOG_DB"))
// 	assert.Equal(t, cfg.Log.Plugin, os.Getenv("XIANG_LOG_PLUGIN"))

// }

// func TestNewConfigFrom(t *testing.T) {
// 	assert.True(t, true)
// 	assert.True(t, true)
// }

func TestLoadFrom(t *testing.T) {
	cfg := LoadFrom(filepath.Join(os.Getenv("YAO_DEV"), ".env"))
	root, _ := filepath.Abs(os.Getenv("YAO_ROOT"))
	assert.Equal(t, cfg.Root, root)
	assert.Equal(t, cfg.Mode, os.Getenv("YAO_ENV"))
	assert.Equal(t, cfg.Host, os.Getenv("YAO_HOST"))
	assert.Equal(t, fmt.Sprintf("%d", cfg.Port), os.Getenv("YAO_PORT"))
	assert.Equal(t, cfg.JWTSecret, os.Getenv("YAO_JWT_SECRET"))
	assert.Equal(t, cfg.Log, os.Getenv("YAO_LOG"))
	assert.Equal(t, cfg.LogMode, os.Getenv("YAO_LOG_MODE"))
	assert.Equal(t, cfg.DB.Driver, os.Getenv("YAO_DB_DRIVER"))
	assert.Equal(t, cfg.DB.Primary[0], os.Getenv("YAO_DB_PRIMARY"))
	// assert.Equal(t, cfg.DB.Secondary[0], os.Getenv("YAO_DB_SECONDARY"))
}
