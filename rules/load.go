package rules

import (
	"fmt"
	"strings"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

var RuleDSLS map[string]*DSL = map[string]*DSL{}

func Load(cfg config.Config) error {
	messages := []string{}
	exts := []string{"*.rul.yao", "*.rul.json", "*.rul.jsonc"}
	err := application.App.Walk("rules", func(root, file string, isdir bool) error {
		if isdir {
			return nil
		}
		if err := LoadFile(root, file); err != nil {
			messages = append(messages, err.Error())
		}

		return nil
	}, exts...)

	exportProcess()

	if len(messages) > 0 {
		return fmt.Errorf(strings.Join(messages, ";\n"))
	}

	return err
}

func LoadFile(root string, file string) error {

	id := share.ID(root, file)
	data, err := application.App.Read(file)
	if err != nil {
		return err
	}

	_, err = load(data, id, file)
	if err != nil {
		return err
	}
	return nil
}

// LoadSource load table dsl by source
func load(source []byte, id string, file string) (*DSL, error) {
	dsl := &DSL{
		Rule: &Rule{
			ID:       id,
			Children: []Rule{},
		},
		file:   file,
		source: source,
	}
	err := dsl.parse()
	if err != nil {
		return nil, fmt.Errorf("[%s] %s", id, err.Error())
	}

	RuleDSLS[id] = dsl
	return dsl, nil
}

// parse method to parse and generate keys
func (dsl *DSL) parse() error {
	err := application.Parse(dsl.file, dsl.source, dsl)
	if err != nil {
		return err
	}
	generateKeys(dsl.Rule, "")
	return nil
}

// generateKeys recursively generates keys for rules
func generateKeys(rule *Rule, parentKey string) {
	if parentKey == "" {
		rule.Key = rule.ID
	} else {
		rule.Key = fmt.Sprintf("%s_%s", parentKey, rule.ID)
	}

	for i := range rule.Children {
		generateKeys(&rule.Children[i], rule.Key)
	}
}

// GetMainKeys returns a slice of main keys in RuleDSLS
func GetMainKeys() []string {
	keys := []string{}
	for key := range RuleDSLS {
		keys = append(keys, key)
	}
	return keys
}

// GetAllKeys 返回Rule结构及其子结构中的所有Key字段
func GetAllKeys() []string {
	var keys []string
	var collectKeys func(r Rule)

	// 定义递归函数来收集Key
	collectKeys = func(r Rule) {
		if r.Key != "" {
			keys = append(keys, r.Key)
		}
		for _, child := range r.Children {
			collectKeys(child)
		}
	}

	// 遍历RuleDSLS，收集所有Key
	for _, dsl := range RuleDSLS {
		if dsl.Rule != nil {
			collectKeys(*dsl.Rule)
		}
	}
	return keys
}
