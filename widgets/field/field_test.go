package field

import (
	"fmt"
	"os"
	"testing"

	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/widgets/expression"
)

func TestLoadAndExport(t *testing.T) {

	// backup YAO_DEV
	dev := os.Getenv("YAO_DEV")

	// From Bindata
	os.Unsetenv("YAO_DEV")
	err := LoadAndExport(config.Conf)
	if err != nil {
		t.Fatal(err)
	}
	if _, has := Transforms["model"]; !has {
		t.Fatal(fmt.Errorf("create model transform error"))
	}

	// clear
	delete(Transforms, "model")

	// From local path
	os.Setenv("YAO_DEV", dev)
	err = LoadAndExport(config.Conf)
	if err != nil {
		t.Fatal(err)
	}
	if _, has := Transforms["model"]; !has {
		t.Fatal(fmt.Errorf("create model transform error"))
	}
}

func testData() map[string]interface{} {
	return maps.MapStr{
		"name":    "Foo",
		"label":   "Bar",
		"comment": "Hi",
		"space":   " Hello World ",
		"variables": map[string]interface{}{
			"color": map[string]interface{}{
				"primary": "#FF0000",
			},
		},
		"option": []interface{}{"Hello", "World"},
	}.Dot()
}

func prepare(t *testing.T) {
	expression.Export()
}
