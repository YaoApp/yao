package component

import (
	"fmt"
	"net/url"
	"strings"

	jsoniter "github.com/json-iterator/go"
	gouProcess "github.com/yaoapp/gou/process"
	"github.com/yaoapp/gou/types"
	"github.com/yaoapp/kun/log"
)

// CloudProps parse CloudProps
func (p PropsDSL) CloudProps(xpath, component string) (map[string]CloudPropsDSL, error) {

	if p == nil {
		return nil, fmt.Errorf("props is required")
	}

	return p.parseCloudProps(xpath, component, p, p)
}

// Path api path
func (cProp CloudPropsDSL) Path() string {
	return fmt.Sprintf("/component/%s/%s", url.QueryEscape(cProp.Xpath), url.QueryEscape(cProp.Name))
}

// UploadPath api UploadPath
func (cProp CloudPropsDSL) UploadPath() string {
	return fmt.Sprintf("/upload/%s/%s", url.QueryEscape(cProp.Xpath), url.QueryEscape(cProp.Name))
}

// ExecUpload execute upload
func (cProp CloudPropsDSL) ExecUpload(process *gouProcess.Process, upload types.UploadFile) (interface{}, error) {

	if upload.TempFile == "" {
		log.Error("[component] %s.$%s upload file is required", cProp.Xpath, cProp.Name)
		return nil, fmt.Errorf("[component] %s.$%s upload file is required", cProp.Xpath, cProp.Name)
	}

	// Process
	name := cProp.Process
	if name == "" {
		log.Error("[component] %s.$%s process is required", cProp.Xpath, cProp.Name)
		return nil, fmt.Errorf("[component] %s.$%s process is required", cProp.Xpath, cProp.Name)
	}

	// Create process
	p, err := gouProcess.Of(name, upload, cProp.Props)
	if err != nil {
		log.Error("[component] %s.$%s %s", cProp.Xpath, cProp.Name, err.Error())
		return nil, fmt.Errorf("[component] %s.$%s %s", cProp.Xpath, cProp.Name, err.Error())
	}

	// Excute process
	err = p.WithGlobal(process.Global).WithSID(process.Sid).Execute()
	if err != nil {
		log.Error("[component] %s.$%s %s", cProp.Xpath, cProp.Name, err.Error())
		return nil, fmt.Errorf("[component] %s.$%s %s", cProp.Xpath, cProp.Name, err.Error())
	}
	defer p.Release()
	res := p.Value()

	return res, nil
}

// ExecQuery execute query
func (cProp CloudPropsDSL) ExecQuery(process *gouProcess.Process, query map[string]interface{}) (interface{}, error) {

	if query == nil {
		query = map[string]interface{}{}
	}

	// filter array
	for key, value := range query {
		if strings.HasSuffix(key, "[]") {
			query[strings.TrimSuffix(key, "[]")] = value
			delete(query, key)
		}
	}

	// Process
	name := cProp.Process
	if name == "" {
		log.Error("[component] %s.$%s process is required", cProp.Xpath, cProp.Name)
		return nil, fmt.Errorf("[component] %s.$%s process is required", cProp.Xpath, cProp.Name)
	}

	// Create process
	p, err := gouProcess.Of(name, query, cProp.Props)
	if err != nil {
		log.Error("[component] %s.$%s %s", cProp.Xpath, cProp.Name, err.Error())
		return nil, fmt.Errorf("[component] %s.$%s %s", cProp.Xpath, cProp.Name, err.Error())
	}

	// Excute process
	err = p.WithGlobal(process.Global).WithSID(process.Sid).Execute()
	if err != nil {
		log.Error("[component] %s.$%s %s", cProp.Xpath, cProp.Name, err.Error())
		return nil, fmt.Errorf("[component] %s.$%s %s", cProp.Xpath, cProp.Name, err.Error())
	}
	defer p.Release()

	res := p.Value()
	return res, nil
}

// Replace xpath
func (cProp CloudPropsDSL) Replace(data interface{}, replace func(cProp CloudPropsDSL) interface{}) error {
	return cProp.replaceAny(data, "", replace)
}

func (cProp CloudPropsDSL) replaceAny(data interface{}, root string, replace func(cProp CloudPropsDSL) interface{}) error {
	switch data.(type) {
	case map[string]interface{}:
		return cProp.replaceMap(data.(map[string]interface{}), root, replace)
	}
	return nil
}

func (cProp CloudPropsDSL) replaceMap(data map[string]interface{}, root string, replace func(cProp CloudPropsDSL) interface{}) error {
	xpath := fmt.Sprintf(".%s.$%s", cProp.Xpath, cProp.Name)

	// get keys
	keys := []string{}
	for key := range data {
		keys = append(keys, key)
	}

	for _, key := range keys {
		path := fmt.Sprintf("%s.%s", root, key)
		if !strings.HasPrefix(xpath, path) {
			continue
		}

		// Replace field
		if path == xpath {
			data[cProp.Name] = replace(cProp)
			delete(data, fmt.Sprintf("$%s", cProp.Name))
			continue
		}

		err := cProp.replaceAny(data[key], path, replace)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p PropsDSL) parseCloudProps(xpath string, component string, props map[string]interface{}, root map[string]interface{}) (map[string]CloudPropsDSL, error) {

	res := map[string]CloudPropsDSL{}

	for name, prop := range props {

		fullname := fmt.Sprintf("%s.%s", xpath, name)
		if sub, ok := prop.(map[string]interface{}); ok {
			cProps, err := p.parseCloudProps(fullname, component, sub, root)
			if err != nil {
				return nil, err
			}
			for k, v := range cProps {
				res[k] = v
			}
		}

		if !strings.HasPrefix(name, "$") {
			continue
		}

		cProp := &CloudPropsDSL{
			Name:  strings.TrimPrefix(name, "$"),
			Type:  component,
			Xpath: xpath,
			Props: root,
		}

		err := cProp.Parse(prop)
		if err != nil {
			return nil, fmt.Errorf("%s %s", fullname, err.Error())
		}

		cProp.Xpath = xpath
		res[fullname] = *cProp
	}

	return res, nil
}

// Has check if the prop exists in the props
func (p PropsDSL) Has(name string) bool {
	_, has := p[name]
	if has {
		return has
	}

	// check if the prop is a cloud prop
	_, has = p[fmt.Sprintf("$%s", name)]
	return has
}

// Parse parse cloud props
func (cProp *CloudPropsDSL) Parse(v interface{}) error {

	bytes, err := jsoniter.Marshal(v)
	if err != nil {
		return err
	}

	err = jsoniter.Unmarshal(bytes, cProp)
	if err != nil {
		return err
	}
	return nil
}
