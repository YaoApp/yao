package environment

import (
	"path/filepath"
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/config"
)

func TestReadFile(t *testing.T) {
	file := filepath.Join(config.Conf.Root, "tables", "env.tab.json")
	defaults := map[string]interface{}{"PAGE_STATUS": "cured"}
	content, err := ReadFile(file, defaults)
	if err != nil {
		t.Fatal(err)
	}

	res := maps.MapStr{}
	err = jsoniter.Unmarshal(content, &res)
	if err != nil {
		t.Fatal(err)
	}

	res = res.Dot()
	// utils.Dump(res)

	assert.Equal(t, "::PET ADMIN", res.Get("name"))
	assert.Equal(t, "20", res.Get("action.search.default.2"))
	assert.Equal(t, "https://yaoapps.com", res.Get("layout.header.preset.import.operation.0.link"))
	assert.Equal(t, "cured", res.Get("layout.table.operation.actions[2].action.Table.save.status"))
	assert.Equal(t, "提示", res["layout.table.operation.actions[5].confirm.title"])
	assert.Equal(t, "icon-trash", res["layout.table.operation.actions[5].icon"])
}
