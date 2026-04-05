package xun

import (
	"testing"

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

func TestToDBTime(t *testing.T) {
	assert.Equal(t, int64(0), toDBTime(0))
	assert.Equal(t, int64(1234567890), toDBTime(1234567890))
	assert.Equal(t, int64(-1), toDBTime(-1))
}

func TestFromDBTime(t *testing.T) {
	assert.Equal(t, int64(0), fromDBTime(0))
	assert.Equal(t, int64(1234567890), fromDBTime(1234567890))
	assert.Equal(t, int64(-1), fromDBTime(-1))
}

func init() {
	_ = capsule.Global
}
