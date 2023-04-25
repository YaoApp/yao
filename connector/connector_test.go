package connector

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/kun/utils"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

func TestLoad(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	err := Load(config.Conf)
	utils.Dump(config.Conf, "ERROR---", err, "-- END ERROR---")
	utils.Dump(
		"REDIS---",
		os.Getenv("REDIS_TEST_HOST"),
		os.Getenv("REDIS_TEST_PORT"),
		os.Getenv("REDIS_TEST_USER"),
		os.Getenv("REDIS_TEST_PASS"),
		"-- END REDIS---",
	)

	utils.Dump(
		"SQLITE---",
		os.Getenv("SQLITE_DB"),
		"-- END SQLITE---",
	)

	if err != nil {
		t.Fatal(err)
	}
	check(t)
}

func check(t *testing.T) {
	ids := map[string]bool{}
	for id := range connector.Connectors {
		ids[id] = true
	}

	utils.Dump(ids)

	assert.True(t, ids["mongo"])
	assert.True(t, ids["mysql"])
	assert.True(t, ids["redis"])
	assert.True(t, ids["sqlite"])
}
