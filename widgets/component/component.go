package component

import (
	"strings"

	jsoniter "github.com/json-iterator/go"
)

// process
// yao.component.TagView
// yao.component.TagEdit
// yao.component.ImageView
// yao.component.UploadEdit

// BackendOnlyProps The component’s properties include visibility for backend only
var BackendOnlyProps = map[string]map[string]map[string]interface{}{
	"select": {
		"query": {
			"xProps": map[string]interface{}{
				"$remote": map[string]interface{}{"process": "yao.component.GetOptions"},
			},
		},
	},
	"autocomplete": {"query": {
		"xProps": map[string]interface{}{
			"$remote": map[string]interface{}{"process": "yao.component.GetOptions"},
		},
	}},
}

// DefaultProps The default properties for the component
var DefaultProps = map[string]map[string]map[string]interface{}{
	"upload": {"api": {"$api": map[string]interface{}{"process": "fs.data.Upload"}}},
	"image":  {"api": {"$api": map[string]interface{}{"process": "utils.throw.Forbidden"}}}, // Just generate an effective URL, no need to upload
}

// UploadComponents the components that need to upload files
var UploadComponents = map[string]bool{
	"upload":     true,
	"wangeditor": true,
	"image":      true,
}

// Export processes
func Export() error {
	exportProcess()
	return nil
}

// MarshalJSON  Custom JSON parse
func (dsl DSL) MarshalJSON() ([]byte, error) {
	return jsoniter.Marshal(dsl.Map())
}

// Map cast to map[string]interface{}
func (dsl DSL) Map() map[string]interface{} {
	res := map[string]interface{}{
		"type":  dsl.Type,
		"props": dsl.FontendProps(),
	}

	if dsl.HideLabel {
		res["hideLabel"] = true
	}

	if dsl.Bind != "" {
		res["bind"] = dsl.Bind
	}
	return res
}

// FontendProps filter backend only properties
func (dsl DSL) FontendProps() map[string]interface{} {
	if dsl.Props == nil {
		return map[string]interface{}{}
	}

	props := map[string]interface{}{}
	t := strings.ToLower(dsl.Type)
	for key, val := range dsl.Props {
		if BackendOnlyProps[t] != nil && BackendOnlyProps[t][key] != nil {
			continue
		}
		props[key] = val
	}
	return props
}

// Parse the component properties
func (dsl *DSL) Parse() {
	t := strings.ToLower(dsl.Type)
	// Check if the component has default props
	if dsl.Props != nil && DefaultProps[t] != nil {
		for key, val := range DefaultProps[t] {
			if !dsl.Props.Has(key) {
				for k, v := range val {
					dsl.Props[k] = v
				}
			}
		}
	}

	// Check if the component has backend only props
	if dsl.Props != nil && BackendOnlyProps[t] != nil {
		for key, val := range BackendOnlyProps[t] {
			if dsl.Props.Has(key) {
				for k, v := range val {
					dsl.Props[k] = v
				}
			}
		}
	}
}

// Clone Component
func (dsl *DSL) Clone() *DSL {
	new := DSL{
		Bind:    dsl.Bind,
		Type:    dsl.Type,
		Compute: dsl.Compute,
		Props:   PropsDSL{},
	}
	if dsl.Props != nil {
		for key, val := range dsl.Props {
			new.Props[key] = val
		}
	}
	return &new
}
