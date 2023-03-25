package field

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/data"
)

// LoadAndExport load table
func LoadAndExport(cfg config.Config) error {

	if os.Getenv("YAO_DEV") != "" {
		file := filepath.Join(os.Getenv("YAO_DEV"), "yao", "fields", "model.trans.json")
		source, err := ioutil.ReadFile(file)
		if err != nil {
			return err
		}
		_, err = OpenTransform(source, "model")
		if err != nil {
			return err
		}
	}

	source, err := data.Read(filepath.Join("yao", "fields", "model.trans.json"))
	if err != nil {
		return err
	}

	_, err = OpenTransform(source, "model")
	if err != nil {
		return err
	}

	return nil
}

// SelectTransform select a transform via name
func SelectTransform(name string) (*Transform, error) {
	trans, has := Transforms[name]
	if !has {
		return nil, fmt.Errorf("Transform %s does not found", name)
	}
	return trans, nil
}

// ModelTransform select model transform via name
func ModelTransform() (*Transform, error) {
	return SelectTransform("model")
}
