package assistant

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/fs"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/yao/neo/store"
	"github.com/yaoapp/yao/share"
	"gopkg.in/yaml.v3"
)

// loaded the loaded assistant
var loaded = NewCache(200) // 200 is the default capacity
var storage store.Store = nil

// SetStorage set the storage
func SetStorage(s store.Store) {
	storage = s
}

// SetCache set the cache
func SetCache(capacity int) {
	ClearCache()
	loaded = NewCache(capacity)
}

// ClearCache clear the cache
func ClearCache() {
	if loaded != nil {
		loaded.Clear()
		loaded = nil
	}
}

// LoadStore create a new assistant from store
func LoadStore(id string) (*Assistant, error) {
	assistant, exists := loaded.Get(id)
	if exists {
		return assistant, nil
	}

	if storage == nil {
		return nil, fmt.Errorf("storage is not set")
	}

	data, err := storage.GetAssistant(id)
	if err != nil {
		return nil, err
	}

	assistant, err = loadMap(data)
	if err != nil {
		return nil, err
	}

	loaded.Put(assistant)
	return assistant, nil
}

// LoadPath load assistant from path
func LoadPath(path string) (*Assistant, error) {
	app, err := fs.Get("app")
	if err != nil {
		return nil, err
	}

	pkgfile := filepath.Join(path, "package.yao")
	if has, _ := app.Exists(pkgfile); !has {
		return nil, fmt.Errorf("package.yao not found in %s", path)
	}

	pkg, err := app.ReadFile(pkgfile)
	if err != nil {
		return nil, err
	}

	id := strings.ReplaceAll(strings.TrimPrefix(path, "/assistants/"), "/", ".")
	var data map[string]interface{}
	err = jsoniter.Unmarshal(pkg, &data)
	if err != nil {
		return nil, err
	}

	// assistant_id
	data["assistant_id"] = id

	// prompts
	promptsfile := filepath.Join(path, "prompts.yml")
	if has, _ := app.Exists(promptsfile); has {
		prompts, err := loadPrompts(promptsfile, path)
		if err != nil {
			return nil, err
		}
		data["prompts"] = prompts
	}

	// load script
	scriptfile := filepath.Join(path, "src", "index.ts")
	if has, _ := app.Exists(scriptfile); has {
		script, err := loadScript(scriptfile, path)
		if err != nil {
			return nil, err
		}
		data["script"] = script
	}

	// load functions

	// load flow

	return loadMap(data)
}

func loadMap(data map[string]interface{}) (*Assistant, error) {

	assistant := &Assistant{}

	// assistant_id is required
	id, ok := data["assistant_id"].(string)
	if !ok {
		return nil, fmt.Errorf("assistant_id is required")
	}
	assistant.ID = id

	// name is required
	name, ok := data["name"].(string)
	if !ok {
		return nil, fmt.Errorf("name is required")
	}
	assistant.Name = name

	// avatar
	if avatar, ok := data["avatar"].(string); ok {
		assistant.Avatar = avatar
	}

	// connector
	if connector, ok := data["connector"].(string); ok {
		assistant.Connector = connector
	}

	// prompts
	if v, ok := data["prompts"].(string); ok {
		var prompts []Prompt
		err := yaml.Unmarshal([]byte(v), &prompts)
		if err != nil {
			return nil, err
		}
		assistant.Prompts = prompts
	}

	// script
	if data["script"] != nil {
		switch v := data["script"].(type) {
		case string:
			file := fmt.Sprintf("assistants/%s/src/index.ts", assistant.ID)
			script, err := loadScriptSource(v, file)
			if err != nil {
				return nil, err
			}
			assistant.Script = script
		case *v8.Script:
			assistant.Script = v
		}
	}

	return assistant, nil
}

func loadPrompts(file string, root string) (string, error) {

	app, err := fs.Get("app")
	if err != nil {
		return "", err
	}

	prompts, err := app.ReadFile(file)
	if err != nil {
		return "", err
	}

	re := regexp.MustCompile(`@assets/([^\s]+\.(md|yml|yaml|json|txt))`)
	prompts = re.ReplaceAllFunc(prompts, func(s []byte) []byte {
		asset := re.FindStringSubmatch(string(s))[1]
		assetFile := filepath.Join(root, "assets", asset)
		assetContent, err := app.ReadFile(assetFile)
		if err != nil {
			return []byte("")
		}
		// Add proper YAML formatting for content
		lines := strings.Split(string(assetContent), "\n")
		formattedContent := "|\n"
		for _, line := range lines {
			formattedContent += "    " + line + "\n"
		}
		return []byte(formattedContent)
	})

	return string(prompts), nil
}

func loadScript(file string, root string) (*v8.Script, error) {
	return v8.Load(file, share.ID(root, file))
}

func loadScriptSource(source string, file string) (*v8.Script, error) {
	script, err := v8.MakeScript([]byte(source), file, 5*time.Second, true)
	if err != nil {
		return nil, err
	}
	return script, nil
}
