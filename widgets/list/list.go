package list

import (
	"fmt"
	"path/filepath"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
	"github.com/yaoapp/yao/widgets/component"
	"github.com/yaoapp/yao/widgets/environment"
	"github.com/yaoapp/yao/widgets/field"
)

//
// API:
//   GET  /api/__yao/list/:id/setting  						-> Default process: yao.list.Xgen
//   GET  /api/__yao/list/:id/get 							-> Default process: yao.list.Get $param.id :query
//   GET  /api/__yao/list/:id/component/:xpath/:method  	-> Default process: yao.list.Component $param.id $param.xpath $param.method :query
//  POST  /api/__yao/list/:id/save  						-> Default process: yao.list.Save $param.id :payload
//   GET  /api/__yao/list/:id/upload/:xpath/:method  		-> Default process: yao.list.Upload $param.id $param.xpath $param.method $file.file
//   GET  /api/__yao/list/:id/download/:field  				-> Default process: yao.list.Download $param.id $param.field $query.name $query.token
//
// Process:
// 	 yao.list.Setting Return the App DSL
// 	 yao.list.Xgen Return the Xgen setting
//   yao.list.Component Return the result defined in props.xProps
//   yao.list.Upload Upload file defined in props
//   yao.list.Download Download file defined in props
//   yao.list.Get Return the query record
//   yao.list.Save Save a record

//
// Hook:
//   before:get
//   after:get
//   before:save
//   after:save

// Lists the loaded list widgets
var Lists map[string]*DSL = map[string]*DSL{}

// New create a new DSL
func New(id string) *DSL {
	return &DSL{
		ID:     id,
		Fields: &FieldsDSL{List: field.Columns{}},
		Layout: &LayoutDSL{},
		CProps: field.CloudProps{},
		Config: map[string]interface{}{},
	}
}

// LoadAndExport load list
func LoadAndExport(cfg config.Config) error {
	err := Load(cfg)
	if err != nil {
		log.Error(err.Error())
	}
	return Export()
}

// Load load task
func Load(cfg config.Config) error {
	var root = filepath.Join(cfg.Root, "lists")
	return LoadFrom(root, "")
}

// LoadFrom load from dir
func LoadFrom(dir string, prefix string) error {

	if share.DirNotExists(dir) {
		return fmt.Errorf("%s does not exists", dir)
	}

	messages := []string{}
	err := share.Walk(dir, ".json", func(root, filename string) {
		id := prefix + share.ID(root, filename)
		data, err := environment.ReadFile(filename)
		if err != nil {
			messages = append(messages, err.Error())
			return
		}

		err = LoadData(data, id, filepath.Dir(dir))
		if err != nil {
			messages = append(messages, err.Error())
		}
	})

	if len(messages) > 0 {
		return fmt.Errorf(strings.Join(messages, ";\n"))
	}

	return err
}

// LoadID load via id
func LoadID(id string, root string) error {
	dirs := strings.Split(id, ".")
	name := fmt.Sprintf("%s.list.json", dirs[len(dirs)-1])
	elems := []string{root}
	elems = append(elems, dirs[0:len(dirs)-1]...)
	elems = append(elems, "lists", name)
	filename := filepath.Join(elems...)
	data, err := environment.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("[List] LoadID %s root=%s %s", id, root, err.Error())
	}
	return LoadData(data, id, root)
}

// LoadData load via data
func LoadData(data []byte, id string, root string) error {
	dsl := New(id)
	dsl.Root = root

	err := jsoniter.Unmarshal(data, dsl)
	if err != nil {
		return fmt.Errorf("[List] LoadData %s %s", id, err.Error())
	}

	if dsl.Action == nil {
		dsl.Action = &ActionDSL{}
	}
	dsl.Action.SetDefaultProcess()

	if dsl.Layout == nil {
		dsl.Layout = &LayoutDSL{}
	}

	if dsl.Fields == nil {
		dsl.Fields = &FieldsDSL{}
	}

	// Bind model / store / list / ...
	err = dsl.Bind()
	if err != nil {
		return fmt.Errorf("[List] LoadData Bind %s %s", id, err.Error())
	}

	// Mapping
	err = dsl.mapping()
	if err != nil {
		return fmt.Errorf("[List] LoadData Mapping %s %s", id, err.Error())
	}

	// Validate
	err = dsl.Validate()
	if err != nil {
		return fmt.Errorf("[List] LoadData Validate %s %s", id, err.Error())
	}

	Lists[id] = dsl
	return nil
}

// Get list via process or id
func Get(list interface{}) (*DSL, error) {
	id := ""
	switch list.(type) {
	case string:
		id = list.(string)
	case *gou.Process:
		id = list.(*gou.Process).ArgsString(0)
	default:
		return nil, fmt.Errorf("%v type does not support", list)
	}

	t, has := Lists[id]
	if !has {
		return nil, fmt.Errorf("%s does not exist", id)
	}
	return t, nil
}

// MustGet Get list via process or id thow error
func MustGet(list interface{}) *DSL {
	t, err := Get(list)
	if err != nil {
		exception.New(err.Error(), 400).Throw()
	}
	return t
}

// Xgen trans to xgen setting
func (dsl *DSL) Xgen(data map[string]interface{}, excludes map[string]bool) (map[string]interface{}, error) {

	if dsl.Layout == nil {
		dsl.Layout = &LayoutDSL{List: &ViewLayoutDSL{}}
	}

	if dsl.Layout.List == nil {
		dsl.Layout.List = &ViewLayoutDSL{}
	}

	layout, err := dsl.Layout.Xgen(data, excludes, dsl.Mapping)
	if err != nil {
		return nil, err
	}

	fields, err := dsl.Fields.Xgen(layout)
	if err != nil {
		return nil, err
	}

	// full width default value
	if _, has := dsl.Config["full"]; !has {
		dsl.Config["full"] = true
	}

	setting := map[string]interface{}{}
	bytes, err := jsoniter.Marshal(layout)
	if err != nil {
		return nil, err
	}

	err = jsoniter.Unmarshal(bytes, &setting)
	if err != nil {
		return nil, err
	}

	onChange := map[string]interface{}{} // Hooks
	setting["fields"] = fields
	setting["config"] = dsl.Config
	for _, cProp := range dsl.CProps {
		err := cProp.Replace(setting, func(cProp component.CloudPropsDSL) interface{} {

			if cProp.Type == "Upload" {
				return fmt.Sprintf("/api/__yao/list/%s%s", dsl.ID, cProp.UploadPath())
			}

			return map[string]interface{}{
				"api":    fmt.Sprintf("/api/__yao/list/%s%s", dsl.ID, cProp.Path()),
				"params": cProp.Query,
			}
		})

		if err != nil {
			return nil, err
		}

		// hooks
		if cProp.Name == "on:change" {
			field := strings.TrimPrefix(cProp.Xpath, "fields.list.")
			field = strings.TrimSuffix(field, ".edit.props")
			onChange[field] = map[string]interface{}{
				"api":    fmt.Sprintf("/api/__yao/list/%s%s", dsl.ID, cProp.Path()),
				"params": cProp.Query,
			}
		}
	}
	setting["hooks"] = map[string]interface{}{"onChange": onChange}
	setting["name"] = dsl.Name
	return setting, nil
}
