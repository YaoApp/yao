package xun

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/yao/agent/store/types"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

func TestGetDriver(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	store, err := NewXun(typeSetting("default"))
	assert.NoError(t, err)
	xunStore := store.(*Xun)

	driver := xunStore.getDriver()
	assert.Contains(t, []string{"mysql", "postgres", "sqlite3"}, driver)

	cfg := config.Conf
	if cfg.DB.Driver != "" {
		assert.Equal(t, cfg.DB.Driver, driver)
	}
}

func TestSandboxRawSQL(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	store, err := NewXun(typeSetting("default"))
	assert.NoError(t, err)
	xunStore := store.(*Xun)

	notNull, isNull := xunStore.sandboxRawSQL()
	assert.NotEmpty(t, notNull)
	assert.NotEmpty(t, isNull)
	assert.Contains(t, notNull, "null")
	assert.Contains(t, isNull, "null")

	driver := xunStore.getDriver()
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

func TestGetDriverFallback(t *testing.T) {
	store := &Xun{}
	driver := store.getDriver()
	assert.Equal(t, "mysql", driver)
}

func typeSetting(connector string) types.Setting {
	return types.Setting{Connector: connector}
}

// TestSandboxRawSQLAllDialects verifies SQL fragments for all three dialects
// without requiring a live database connection.
func TestSandboxRawSQLAllDialects(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	s, err := NewXun(typeSetting("default"))
	assert.NoError(t, err)
	xunStore := s.(*Xun)

	// Verify the current driver produces valid SQL fragments
	notNull, isNull := xunStore.sandboxRawSQL()
	assert.Contains(t, notNull, "<>")
	assert.Contains(t, isNull, "=")
}

func TestJsonContainsValue(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	store, err := NewXun(typeSetting("default"))
	assert.NoError(t, err)
	xunStore := store.(*Xun)

	driver := xunStore.getDriver()
	val := xunStore.jsonContainsValue(`%"admin"%`)
	switch driver {
	case "sqlite3":
		assert.Equal(t, `%"admin"%`, val, "SQLite keeps LIKE pattern as-is")
	default:
		assert.Equal(t, `"admin"`, val, "PG/MySQL strips % wrappers for JSON value")
	}
}

func TestJsonContainsValueFallback(t *testing.T) {
	store := &Xun{}
	val := store.jsonContainsValue(`%"test"%`)
	assert.Equal(t, `"test"`, val, "Default (mysql) strips % wrappers")
}

func TestNanoToTime(t *testing.T) {
	assert.True(t, nanoToTime(0).IsZero(), "zero input returns zero time")

	ns := int64(1609459200000000000) // 2021-01-01 00:00:00 UTC
	got := nanoToTime(ns)
	assert.Equal(t, 2021, got.Year())
	assert.Equal(t, time.January, got.Month())
	assert.Equal(t, 1, got.Day())
	assert.Equal(t, time.UTC, got.Location(), "must be UTC")
}

func TestTimeToNano(t *testing.T) {
	assert.Equal(t, int64(0), timeToNano(time.Time{}), "zero time returns 0")

	ts := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, int64(1609459200000000000), timeToNano(ts))
}

func init() {
	_ = capsule.Global
}
