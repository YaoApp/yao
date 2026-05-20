//go:build integration

package xun_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/store/types"
	"github.com/yaoapp/yao/agent/store/xun"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestGetDriver(t *testing.T) {
	testprepare.PrepareSandbox(t)

	store, err := xun.NewXun(types.Setting{Connector: "default"})
	require.NoError(t, err)

	xunStore := store.(*xun.Xun)
	driver := xun.GetDriverForTest(xunStore)
	assert.Contains(t, []string{"mysql", "postgres", "sqlite3"}, driver)
}

func TestSandboxRawSQL(t *testing.T) {
	testprepare.PrepareSandbox(t)

	store, err := xun.NewXun(types.Setting{Connector: "default"})
	require.NoError(t, err)

	xunStore := store.(*xun.Xun)
	notNull, isNull := xun.SandboxRawSQLForTest(xunStore)
	assert.NotEmpty(t, notNull)
	assert.NotEmpty(t, isNull)
	assert.Contains(t, notNull, "null")
	assert.Contains(t, isNull, "null")

	driver := xun.GetDriverForTest(xunStore)
	switch driver {
	case "postgres":
		assert.Contains(t, notNull, `"sandbox"::text`)
		assert.Contains(t, isNull, `"sandbox"::text`)
	case "sqlite3":
		assert.Contains(t, notNull, "CAST(sandbox AS TEXT)")
		assert.Contains(t, isNull, "CAST(sandbox AS TEXT)")
	default:
		assert.Contains(t, notNull, "CAST(`sandbox` AS CHAR)")
		assert.Contains(t, isNull, "CAST(`sandbox` AS CHAR)")
	}
}
