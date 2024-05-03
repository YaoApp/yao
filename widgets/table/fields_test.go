package table

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

func TestFiledsBindModel(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	prepare(t)
	clear(t)
	testData(t)

	m := model.Select("pet")
	tab := New("unit-test", "unit-test.tab.yao", nil)
	err := tab.Fields.BindModel(m)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "id", tab.Fields.Table["ID"].Bind)
	assert.Equal(t, 18, len(tab.Fields.Table))
	assert.Equal(t, 7, len(tab.Fields.Filter))
}
