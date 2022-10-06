package table

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou"
)

func TestFiledsBindModel(t *testing.T) {
	load(t)
	clear(t)
	testData(t)
	m := gou.Select("pet")
	tab := New("unit-test")
	err := tab.Fields.BindModel(m)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "id", tab.Fields.Table["ID"].Bind)
	assert.Equal(t, 15, len(tab.Fields.Table))
	assert.Equal(t, 4, len(tab.Fields.Filter))
}
