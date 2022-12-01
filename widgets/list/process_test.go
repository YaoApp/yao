package list

import (
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/yao/config"
	q "github.com/yaoapp/yao/query"
)

func TestProcessSetting(t *testing.T) {
	load(t)
	clear(t)
	testData(t)
	args := []interface{}{"category"}
	res, err := gou.NewProcess("yao.list.Setting", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	data := any.Of(res).MapStr().Dot()
	assert.Equal(t, "/api/__yao/list/category/component/fields.list."+url.QueryEscape("父类")+".edit.props.xProps/remote", data.Get("fields.list.父类.edit.props.xProps.remote.api"))
}

func TestProcessXgen(t *testing.T) {
	load(t)
	clear(t)
	testData(t)
	args := []interface{}{"category"}
	res, err := gou.NewProcess("yao.list.Xgen", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	data := any.Of(res).MapStr().Dot()
	assert.Equal(t, "/api/__yao/list/category/component/fields.list."+url.QueryEscape("父类")+".edit.props.xProps/remote", data.Get("fields.list.父类.edit.props.xProps.remote.api"))
}

func load(t *testing.T) {
	prepare(t)
	err := Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}
	q.Load(config.Conf)
}

func testData(t *testing.T) {
	category := gou.Select("category")
	err := category.Insert(
		[]string{"name", "stock", "status", "rank"},
		[][]interface{}{
			{"机器人", 100, "启用", 1},
			{"运输车", 80, "启用", 2},
			{"货柜", 100, "停用", 3},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
}

func tempFile(t *testing.T) string {
	file, err := os.CreateTemp("", "unit-test")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	_, err = file.Write([]byte("HELLO"))
	if err != nil {
		t.Fatal(err)
	}

	return file.Name()
}

func clear(t *testing.T) {
	for _, m := range gou.Models {
		err := m.DropTable()
		if err != nil {
			t.Fatal(err)
		}
		err = m.Migrate(true)
		if err != nil {
			t.Fatal(err)
		}
	}
}
