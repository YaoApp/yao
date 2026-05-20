//go:build unit

package xun_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/agent/store/xun"
)

func TestGetDriverFallback(t *testing.T) {
	store := xun.EmptyXunForTest()
	driver := xun.GetDriverForTest(store)
	assert.Equal(t, "mysql", driver)
}

func TestJsonContainsValueFallback(t *testing.T) {
	store := xun.EmptyXunForTest()
	val := xun.JsonContainsValueForTest(store, `%"test"%`)
	assert.Equal(t, `"test"`, val, "Default (mysql) strips %% wrappers")
}
