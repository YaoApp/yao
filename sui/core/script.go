package core

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/application"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/yao/share"
)

// Scripts loaded scripts
var Scripts = map[string]*Script{}

const (
	saveScript uint8 = iota
	removeScript
)

// Script the script
type Script struct {
	*v8.Script
}

type scriptData struct {
	file   string
	script *Script
	cmd    uint8
}

var chScript = make(chan *scriptData, 1)

func init() {
	go scriptWriter()
}

func scriptWriter() {
	for {
		select {
		case data := <-chScript:
			switch data.cmd {
			case saveScript:
				Scripts[data.file] = data.script
			case removeScript:
				delete(Scripts, data.file)
			}
		}
	}
}

// LoadScript load the script
func LoadScript(file string, disableCache ...bool) (*Script, error) {

	base := strings.TrimSuffix(strings.TrimSuffix(file, ".sui"), ".jit")
	// LOAD FROM CACHE
	if disableCache == nil || !disableCache[0] {
		if script, has := Scripts[base]; has {
			return script, nil
		}
	}

	file = base + ".backend.ts"
	if exist, _ := application.App.Exists(file); !exist {
		file = base + ".backend.js"
	}

	if exist, _ := application.App.Exists(file); !exist {
		return nil, nil
	}

	source, err := application.App.Read(file)
	if err != nil {
		return nil, err
	}

	v8script, err := v8.MakeScript(source, file, 5*time.Second)
	if err != nil {
		return nil, err
	}

	v8script.SourceRoots = getSourceRootReplaceFunc()
	script := &Script{Script: v8script}
	chScript <- &scriptData{base, script, saveScript}
	return script, nil
}

// Call the script method
// This will be refactored to improve the performance
func (script *Script) Call(r *Request, method string, args ...any) (interface{}, error) {
	ctx, err := script.NewContext(r.Sid, nil)
	if err != nil {
		return nil, err
	}
	defer ctx.Close()
	if args == nil {
		args = []any{}
	}
	args = append(args, r)

	// Set the sid
	ctx.Sid = r.Sid
	res, err := ctx.Call(method, args...)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// BeforeRender the script method
func (script *Script) BeforeRender(r *Request, props map[string]interface{}) (Data, error) {

	ctx, err := script.NewContext(r.Sid, nil)
	if err != nil {
		return nil, err
	}
	defer ctx.Close()

	if !ctx.Global().Has("BeforeRender") {
		return nil, nil
	}

	// Set the sid
	ctx.Sid = r.Sid
	res, err := ctx.Call("BeforeRender", r, props)
	if err != nil {
		return nil, err
	}

	if data, ok := res.(map[string]interface{}); ok {
		return data, nil
	}

	return nil, fmt.Errorf("BeforeRender return %v should be Record<string, any>", res)
}

// ConstantsToString get the constants from the script
func (script *Script) ConstantsToString() (string, error) {
	constants, err := script.Constants()
	if err != nil {
		return "", err
	}
	raw, err := jsoniter.MarshalToString(constants)
	if err != nil {
		return "", err
	}
	return raw, nil
}

// Constants  get the constants from the script
// This will be refactored to improve the performance
func (script *Script) Constants() (map[string]interface{}, error) {
	uuid := uuid.New().String()
	ctx, err := script.NewContext(uuid, nil)
	if err != nil {
		return nil, err
	}
	defer ctx.Close()

	global := ctx.Global()
	if global == nil {
		return nil, fmt.Errorf("global is nil")
	}

	if !global.Has("__sui_constants") {
		return nil, nil
	}

	res, err := global.Get("__sui_constants")
	if err != nil {
		return nil, err
	}
	defer res.Release()

	goValues, err := bridge.GoValue(res, ctx.Context)
	if err != nil {
		return nil, err
	}

	if constants, ok := goValues.(map[string]interface{}); ok {
		return constants, nil
	}

	return nil, fmt.Errorf("constants is %v should be Record<string, any>", goValues)
}

// Helpers get the helpers from the script
// This will be refactored to improve the performance
func (script *Script) Helpers() ([]string, error) {
	uuid := uuid.New().String()
	ctx, err := script.NewContext(uuid, nil)
	if err != nil {
		return nil, err
	}
	defer ctx.Close()

	global := ctx.Global()
	if global == nil {
		return nil, fmt.Errorf("global is nil")
	}

	if !global.Has("__sui_helpers") {
		return nil, nil
	}

	res, err := global.Get("__sui_helpers")
	if err != nil {
		return nil, err
	}
	defer res.Release()

	goValues, err := bridge.GoValue(res, ctx.Context)
	if err != nil {
		return nil, err
	}

	if helpers, ok := goValues.([]interface{}); ok {
		methods := []string{}
		for _, key := range helpers {
			methods = append(methods, fmt.Sprintf("%v", key))
		}
		return methods, nil
	}

	return nil, fmt.Errorf("helpers is %v should be []string", goValues)
}

func getSourceRootReplaceFunc() interface{} {
	if share.App.Static.SourceRoots == nil {
		return nil
	}
	roots := share.App.Static.SourceRoots
	return func(file string) string {
		for name, mapping := range roots {
			if strings.HasPrefix(file, name) {
				path := mapping + strings.TrimPrefix(file, name)
				base := filepath.Base(path)
				name := strings.TrimSuffix(base, ".backend.ts")
				dir := filepath.Dir(path)
				return filepath.Join(dir, name, base)
			}
		}
		return file
	}
}
